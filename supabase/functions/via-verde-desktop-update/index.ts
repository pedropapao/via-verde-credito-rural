import { createClient } from "npm:@supabase/supabase-js@2";

const BUCKET = "via-verde-files";
const ROOT = "desktop-updates";

function json(data: unknown, status = 200) {
  return new Response(JSON.stringify(data), {
    status,
    headers: {
      "content-type": "application/json; charset=utf-8",
      "cache-control": "no-store",
      "access-control-allow-origin": "*",
      "access-control-allow-headers": "authorization,content-type,x-github-run-id,x-github-sha,x-version,x-sha256,x-size,x-notes-base64,x-publisher-mode",
      "access-control-allow-methods": "GET,POST,OPTIONS",
    },
  });
}

function decodeNotes(v: string | null) {
  if (!v) return "";
  try {
    const bytes = Uint8Array.from(atob(v), (c) => c.charCodeAt(0));
    return new TextDecoder().decode(bytes).trim();
  } catch {
    return "";
  }
}

async function readFinalizeNotes(req: Request) {
  const headerNotes = decodeNotes(req.headers.get("x-notes-base64"));
  if (headerNotes) return headerNotes;
  const contentType = (req.headers.get("content-type") || "").toLowerCase();
  if (!contentType.includes("application/json")) return "";
  try {
    const body = await req.json();
    return typeof body?.notes === "string" ? body.notes.trim() : "";
  } catch {
    return "";
  }
}

function adminClient() {
  const url = Deno.env.get("SUPABASE_URL");
  const legacy = Deno.env.get("SUPABASE_SERVICE_ROLE_KEY");
  const newerRaw = Deno.env.get("SUPABASE_SECRET_KEYS");
  let key = legacy || "";
  if (!key && newerRaw) {
    try {
      const parsed = JSON.parse(newerRaw);
      key = parsed.default || Object.values(parsed)[0] || "";
    } catch {}
  }
  if (!url || !key) throw new Error("Credencial administrativa do Storage indisponível");
  return createClient(url, String(key), {
    auth: { persistSession: false, autoRefreshToken: false },
  });
}

const GITHUB_REPO = "pedropapao/via-verde-credito-rural";
const GITHUB_BRANCH = "desktop-v1-car-current";

async function githubActionsPublisherAuthorized(req: Request) {
  const authorization = (req.headers.get("authorization") || "").trim();
  const runId = (req.headers.get("x-github-run-id") || "").trim();
  const sha = (req.headers.get("x-github-sha") || "").trim().toLowerCase();
  if (!authorization.startsWith("Bearer ") || !/^\d+$/.test(runId) || !/^[a-f0-9]{40}$/.test(sha)) {
    return false;
  }
  const resp = await fetch(`https://api.github.com/repos/${GITHUB_REPO}/actions/runs/${runId}`, {
    headers: {
      authorization,
      accept: "application/vnd.github+json",
      "x-github-api-version": "2022-11-28",
      "user-agent": "ViaVerdeCAR-Updater",
    },
  });
  if (!resp.ok) return false;
  const run = await resp.json();
  return run?.repository?.full_name === GITHUB_REPO &&
    String(run?.head_sha || "").toLowerCase() === sha &&
    run?.event === "push" &&
    run?.head_branch === GITHUB_BRANCH;
}


async function githubLocalPublisherAuthorized(req: Request) {
  const mode = (req.headers.get("x-publisher-mode") || "").trim().toLowerCase();
  const authorization = (req.headers.get("authorization") || "").trim();
  const sha = (req.headers.get("x-github-sha") || "").trim().toLowerCase();
  if (mode !== "local" || !authorization.startsWith("Bearer ") || !/^[a-f0-9]{40}$/.test(sha)) {
    return false;
  }

  const headers = {
    authorization,
    accept: "application/vnd.github+json",
    "x-github-api-version": "2022-11-28",
    "user-agent": "ViaVerdeCAR-LocalPublisher",
  };

  const [userResp, repoResp, commitResp] = await Promise.all([
    fetch("https://api.github.com/user", { headers }),
    fetch(`https://api.github.com/repos/${GITHUB_REPO}`, { headers }),
    fetch(`https://api.github.com/repos/${GITHUB_REPO}/commits/${sha}`, { headers }),
  ]);
  if (!userResp.ok || !repoResp.ok || !commitResp.ok) return false;

  const user = await userResp.json();
  const repo = await repoResp.json();
  const expectedOwner = GITHUB_REPO.split("/")[0].toLowerCase();

  return String(user?.login || "").toLowerCase() === expectedOwner &&
    repo?.full_name === GITHUB_REPO &&
    Boolean(repo?.permissions?.admin || repo?.permissions?.maintain || repo?.permissions?.push);
}

async function publisherAuthorized(req: Request) {
  if (await githubActionsPublisherAuthorized(req)) return true;
  return await githubLocalPublisherAuthorized(req);
}

Deno.serve(async (req) => {
  if (req.method === "OPTIONS") return new Response(null, { status: 204 });
  const u = new URL(req.url);
  const action = u.searchParams.get("action") || "manifest";

  try {
    const supabase = adminClient();

    if (req.method === "GET" && action === "manifest") {
      const { data, error } = await supabase.storage.from(BUCKET).download(`${ROOT}/manifest.json`);
      if (error) {
        return json({ version: "0.0.0", available: false, message: "Nenhuma atualização publicada ainda." }, 200);
      }
      const manifest = JSON.parse(await data.text());
      const { data: signed, error: signErr } = await supabase.storage
        .from(BUCKET)
        .createSignedUrl(manifest.path, 60 * 30);
      if (signErr || !signed?.signedUrl) throw signErr || new Error("Não foi possível gerar link temporário");
      return json({ ...manifest, download_url: signed.signedUrl });
    }

    if (req.method === "POST" && action === "prepare") {
      if (!(await publisherAuthorized(req))) {
        return json({ error: "unauthorized publisher" }, 401);
      }
      const version = (req.headers.get("x-version") || "").trim();
      const sha256 = (req.headers.get("x-sha256") || "").trim().toLowerCase();
      const size = Number((req.headers.get("x-size") || "0").trim());
      if (!/^\d+\.\d+\.\d+(?:[-+][0-9A-Za-z.-]+)?$/.test(version)) {
        return json({ error: "invalid version" }, 400);
      }
      if (!/^[a-f0-9]{64}$/.test(sha256)) {
        return json({ error: "invalid sha256" }, 400);
      }
      if (!Number.isFinite(size) || size < 1024 * 1024 || size > 50 * 1024 * 1024) {
        return json({ error: "invalid executable size", size }, 400);
      }
      const path = `${ROOT}/ViaVerdeCAR-${version}.exe`;
      const { data: signed, error: signErr } = await supabase.storage
        .from(BUCKET)
        .createSignedUploadUrl(path, { upsert: true });
      if (signErr || !signed?.signedUrl) {
        throw signErr || new Error("Não foi possível gerar URL assinada de upload");
      }
      return json({ ok: true, path, upload_url: signed.signedUrl });
    }

    if (req.method === "POST" && action === "finalize") {
      if (!(await publisherAuthorized(req))) {
        return json({ error: "unauthorized publisher" }, 401);
      }
      const version = (req.headers.get("x-version") || "").trim();
      const sha256 = (req.headers.get("x-sha256") || "").trim().toLowerCase();
      const size = Number((req.headers.get("x-size") || "0").trim());
      const notes = await readFinalizeNotes(req);
      if (!/^\d+\.\d+\.\d+(?:[-+][0-9A-Za-z.-]+)?$/.test(version)) {
        return json({ error: "invalid version" }, 400);
      }
      if (!/^[a-f0-9]{64}$/.test(sha256)) {
        return json({ error: "invalid sha256" }, 400);
      }
      if (!Number.isFinite(size) || size < 1024 * 1024 || size > 50 * 1024 * 1024) {
        return json({ error: "invalid executable size", size }, 400);
      }

      const fileName = `ViaVerdeCAR-${version}.exe`;
      const path = `${ROOT}/${fileName}`;
      const { data: objects, error: listErr } = await supabase.storage
        .from(BUCKET)
        .list(ROOT, { limit: 100, search: fileName });
      if (listErr) throw listErr;
      const object = (objects || []).find((x) => x.name === fileName);
      if (!object) return json({ error: "uploaded executable not found" }, 409);
      const remoteSize = Number(object?.metadata?.size || 0);
      if (remoteSize > 0 && remoteSize !== size) {
        return json({ error: "uploaded executable size mismatch", expected: size, actual: remoteSize }, 409);
      }

      const manifest = {
        version,
        path,
        sha256,
        size,
        notes: notes || `Via Verde CAR ${version}`,
        published_at: new Date().toISOString(),
        channel: "stable",
      };
      const manifestBody = new TextEncoder().encode(JSON.stringify(manifest));
      const { error: manifestErr } = await supabase.storage
        .from(BUCKET)
        .upload(`${ROOT}/manifest.json`, manifestBody, {
          contentType: "application/json; charset=utf-8",
          cacheControl: "60",
          upsert: true,
        });
      if (manifestErr) throw manifestErr;
      return json({ ok: true, ...manifest });
    }

    if (req.method === "POST" && action === "publish") {
      return json({ error: "legacy publish disabled; use prepare/finalize" }, 410);
    }

    return json({ error: "not found" }, 404);
  } catch (err) {
    return json({ error: String(err?.message || err) }, 500);
  }
});