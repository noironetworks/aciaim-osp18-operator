package ciscoaciaim

const (
    // NeutronUID and NeutronGID are the user and group IDs for the Neutron service.
    NeutronUID int64 = 42435
    NeutronGID int64 = 42435
)


const (
    ConditionDBReady        = "DBReady"
    ConditionRabbitMQReady  = "RabbitMQReady"
    ConditionAimReady       = "AimReady"
)

const (
    // ServiceCommand is the entrypoint command for the AIM service container.
    ServiceCommand = "/usr/local/bin/kolla_start"
)
