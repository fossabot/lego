package statsd

import (
	"fmt"
	"time"

	"github.com/stairlin/lego/config"
	"github.com/stairlin/lego/stats"
)

const Name = "statsd"

const prefix = "lego"

var tagsFormats = map[string]TagFormat{
	"influxdb": InfluxDB,
	"datadog":  Datadog,
}

func New(tree config.Tree) (stats.Stats, error) {
	config := &Config{}
	if err := tree.Unmarshal(config); err != nil {
		return nil, err
	}

	conf := extractConfig(config)
	client := &Client{
		config: conf,
	}

	// Create connection
	conn, err := newConn(conf.Conn, conf.Client.Muted)
	if err != nil {
		client.muted = true

		// If nothing is listening on the target port, an error is returned and
		// the returned client does nothing but is still usable. So we can
		// just log the error and go on.
		return nil, err
	}
	client.conn = conn

	return client, nil
}

type Client struct {
	conn   *conn
	config *adapterConfig
	muted  bool
}

func (c *Client) Start() {

}

func (c *Client) Stop() {
	c.close()
}

func (c *Client) Count(key string, n interface{}, tags ...map[string]string) {
	c.conn.metric(prefix, key, n, "c", c.config.Client.Rate, c.buildTags(tags...))
}

func (c *Client) Inc(key string, tags ...map[string]string) {
	c.Count(key, 1, tags...)
}

func (c *Client) Dec(key string, tags ...map[string]string) {
	c.Count(key, -1, tags...)
}

func (c *Client) Gauge(key string, n interface{}, tags ...map[string]string) {
	c.conn.gauge(prefix, key, n, c.buildTags(tags...))
}

func (c *Client) Timing(key string, t time.Duration, tags ...map[string]string) {
	d := t.Nanoseconds() / 1000000
	c.conn.metric(prefix, key, d, "ms", c.config.Client.Rate, c.buildTags(tags...))
}

func (c *Client) Histogram(key string, n interface{}, tags ...map[string]string) {
	c.conn.metric(prefix, key, n, "h", c.config.Client.Rate, c.buildTags(tags...))
}

func (c *Client) buildTags(l ...map[string]string) string {
	return c.joinTags(c.mergeTags(l...))
}

// joinTags joins tags in a specific tag format (e.g. InfluxDB, Datadog, ...)
func (c *Client) joinTags(tags []tag) string {
	tf := c.config.Conn.TagFormat
	if len(tags) == 0 || tf == 0 {
		return ""
	}

	join := joinFuncs[tf]
	return join(tags)
}

// mergeTags merges global tags with the tags given
func (c *Client) mergeTags(l ...map[string]string) []tag {
	if len(l) == 0 {
		return c.config.Client.Tags
	}

	global := c.config.Client.Tags
	metric := converTags(l[0])

	return append(global, metric...)
}

// close flushes the Client's buffer and releases the associated ressources. The
// Client and all the cloned Clients must not be used afterward.
func (c *Client) close() {
	if c.muted {
		return
	}
	c.conn.mu.Lock()
	c.conn.flush(0)
	c.conn.handleError(c.conn.w.Close())
	c.conn.closed = true
	c.conn.mu.Unlock()
}

func converTags(m map[string]string) []tag {
	var tags []tag
	for k, v := range m {
		tags = append(tags, tag{K: k, V: v})
	}

	return tags
}

func extractConfig(c *Config) *adapterConfig {
	// The default configuration.
	conf := &adapterConfig{
		Client: clientConfig{
			Rate: 1,
		},
		Conn: connConfig{
			Addr:        ":8125",
			FlushPeriod: 100 * time.Millisecond,
			// Worst-case scenario:
			// Ethernet MTU - IPv6 Header - TCP Header = 1500 - 40 - 20 = 1440
			MaxPacketSize: 1440,
			Network:       "udp",
		},
	}

	// Address
	conf.Conn.Addr = fmt.Sprintf("%s:%s", c.Addr, c.Port)

	// Tags format
	if c.TagsFormat != "" {
		conf.Conn.TagFormat = tagsFormats[c.TagsFormat]
	}

	// Global tags
	// They are sent with each metric
	// It can various things, such as node name, datacenter, ...
	for k, v := range c.Tags {
		t := tag{
			K: k,
			V: v,
		}
		conf.Client.Tags = append(conf.Client.Tags, t)

	}

	return conf
}
