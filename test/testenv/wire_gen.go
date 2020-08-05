// Code generated by Wire. DO NOT EDIT.

//go:generate wire
//+build !wireinject

package testenv

import (
	"github.com/google/wire"
	"github.com/redhat-marketplace/redhat-marketplace-operator/pkg/config"
	"github.com/redhat-marketplace/redhat-marketplace-operator/pkg/controller"
	"github.com/redhat-marketplace/redhat-marketplace-operator/pkg/utils/reconcileutils"
)

// Injectors from wire.go:

func initializeLocalSchemes() (controller.LocalSchemes, error) {
	opsSrcSchemeDefinition := controller.ProvideOpsSrcScheme()
	monitoringSchemeDefinition := controller.ProvideMonitoringScheme()
	olmV1SchemeDefinition := controller.ProvideOLMV1Scheme()
	olmV1Alpha1SchemeDefinition := controller.ProvideOLMV1Alpha1Scheme()
	openshiftConfigV1SchemeDefinition := controller.ProvideOpenshiftConfigV1Scheme()
	localSchemes := controller.ProvideLocalSchemes(opsSrcSchemeDefinition, monitoringSchemeDefinition, olmV1SchemeDefinition, olmV1Alpha1SchemeDefinition, openshiftConfigV1SchemeDefinition)
	return localSchemes, nil
}

func initializeControllers() (controller.ControllerList, error) {
	defaultCommandRunnerProvider := reconcileutils.ProvideDefaultCommandRunnerProvider()
	marketplaceController := controller.ProvideMarketplaceController(defaultCommandRunnerProvider)
	meterbaseController := controller.ProvideMeterbaseController(defaultCommandRunnerProvider)
	meterDefinitionController := controller.ProvideMeterDefinitionController(defaultCommandRunnerProvider)
	razeeDeployController := controller.ProvideRazeeDeployController()
	olmSubscriptionController := controller.ProvideOlmSubscriptionController()
	operatorConfig, err := config.ProvideConfig()
	if err != nil {
		return nil, err
	}
	meterReportController := controller.ProvideMeterReportController(defaultCommandRunnerProvider, operatorConfig)
	olmClusterServiceVersionController := controller.ProvideOlmClusterServiceVersionController()
	remoteResourceS3Controller := controller.ProvideRemoteResourceS3Controller()
	nodeController := controller.ProvideNodeController()
	controllerList := controller.ProvideControllerList(marketplaceController, meterbaseController, meterDefinitionController, razeeDeployController, olmSubscriptionController, meterReportController, olmClusterServiceVersionController, remoteResourceS3Controller, nodeController)
	return controllerList, nil
}

// wire.go:

var TestControllerSet = wire.NewSet(controller.ControllerSet, controller.ProvideControllerFlagSet, controller.SchemeDefinitions, config.ProvideConfig, reconcileutils.ProvideDefaultCommandRunnerProvider, wire.Bind(new(reconcileutils.ClientCommandRunnerProvider), new(*reconcileutils.DefaultCommandRunnerProvider)))
