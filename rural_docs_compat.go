package main

// Os testes e partes legadas ainda podem renderizar client_detail com
// ManagerClientDetailView. Estes métodos fornecem os campos novos da central
// documental sem alterar o fluxo antigo; a rota atual usa RuralClientDetailView.
func (v ManagerClientDetailView) PropertyCount() int { return 0 }
func (v ManagerClientDetailView) Properties() []Property { return nil }
