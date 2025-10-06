package main

// Invoice represents a customer invoice
type Invoice struct {
	Subtotal float64
	Tax      float64
	Total    float64
}

// CalculateTax computes the tax for this invoice
func (i *Invoice) CalculateTax() {
	i.Tax = i.Subtotal * 0.08
}

// CalculateTotal computes the total amount
func (i *Invoice) CalculateTotal() {
	i.CalculateTax()
	i.Total = i.Subtotal + i.Tax
}
