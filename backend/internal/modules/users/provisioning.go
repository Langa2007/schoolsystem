package users

type ProvisioningPolicy struct {
	RequiresParentPhone bool
	RequiresNEMISUPI    bool
}

var DefaultProvisioningPolicy = ProvisioningPolicy{RequiresParentPhone: true}
