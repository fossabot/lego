package disco

import (
	"net"
	"strconv"

	"github.com/stairlin/lego/ctx"
)

// An Agent interacts with a service discovery cluster to manage all services
// offered by the local node. It also allows to query the service discovery
// cluster to fetch services offered by other nodes.
type Agent interface {
	// Register adds a new service to the catalogue
	Register(ctx ctx.Ctx, s *Registration) (string, error)
	// Deregister removes a service from the catalogue
	// If the service does not exist, no action is taken.
	Deregister(ctx ctx.Ctx, id string) error
	// Services returns all registered service instances
	Services(ctx ctx.Ctx, tags ...string) (map[string]Service, error)
	// Service returns all instances of a service
	Service(ctx ctx.Ctx, name string, tags ...string) (Service, error)
	// Leave is used to have the agent de-register all services from the catalogue
	// that belong to this node, and gracefully leave
	Leave(ctx ctx.Ctx)
}

// A Service is a set of functionalities offered by one or multiple nodes on
// the network. A node that offers the service is called an instance.
// A node can offer multiple services, so there will be multiple instances on
// the same node.
type Service interface {
	// Name returns the unique name of a service
	Name() string
	// Watch listens to service updates
	Watch() Watcher
	// Instances returns all available instances of the service
	Instances() []*Instance
}

// An Instance is an instance of a remotely-accessible service on the network
type Instance struct {
	// Local tells whether it is a local or remote instance
	Local bool
	// ID is the unique instance identifier
	ID string
	// Name is a friendly name
	Name string
	Host string
	Port uint16
	Tags []string
}

// Addr returns the instance host+port
func (i *Instance) Addr() string {
	return net.JoinHostPort(i.Host, strconv.FormatUint(uint64(i.Port), 10))
}

// Registration allows to register a service
type Registration struct {
	Name string
	Addr string
	Port uint16
	Tags []string
}
