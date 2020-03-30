package controller

import (
	"github.ibm.com/symposium/marketplace-operator/pkg/controller/razeedeployment"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, razeedeployment.Add)
	flagSets = append(flagSets, razeedeployment.FlagSet())
}