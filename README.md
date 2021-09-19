# Renovate-Operator
`RenovateOperator` is an EXPERIMENTAL application written with OperatorSDK to host
RenovateBot inside a Kubernetes cluster. 

Supported features: 
  - Autodiscovery mode 
  - Run on schedule 
  - DryRun mode
  - Suspend execution
  - Shared redis cache
  - Multi Worker setup

Supported platforms:
  - Github
  - Github Enterprise
  - Gitlab.com
  - Gitlab CE & EE


For CRD usage examples see [samples](./config/samples/renovate_v1alpha1_renovate.yaml)
