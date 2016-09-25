package statsd

import (
	"fmt"
	"strings"
	"time"

	"github.com/stairlin/lego/config"
	"github.com/stairlin/lego/log"
	"github.com/stairlin/lego/stats"
)

const Name = "statsd"

const prefix = "lego"

var tagsFormats = map[string]TagFormat{
	"influxdb": InfluxDB,
	"datadog":  Datadog,
}

func New(c map[string]string) (stats.Stats, error) {
	conf := extractConfig(c)
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
	logger log.Logger
	muted  bool
}

func (c *Client) Start() {
	c.logger.Tracef("statsd: connecting to <%s>...", c.conn.addr)
}

func (c *Client) Stop() {
	c.close()
}

func (c *Client) SetLogger(l log.Logger) {
	c.logger = l
}

func (c *Client) Count(key string, n interface{}, tags ...map[string]string) {
	c.conn.metric(prefix, key, n, "c", c.config.Client.Rate, c.buildTags(tags...))
}

func (c *Client) Inc(key string, tags ...map[string]string) {
	c.Count(key, 1)
}

func (c *Client) Dec(key string, tags ...map[string]string) {
	c.Count(key, -1)
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

func extractConfig(c map[string]string) *adapterConfig {
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
	if _, ok := c["port"]; ok {
		addr := config.ValueOf(c["addr"])
		port := config.ValueOf(c["port"])

		conf.Conn.Addr = fmt.Sprintf("%s:%s", addr, port)
	}

	// Tags format
	if c["tags_format"] != "" {
		conf.Conn.TagFormat = tagsFormats[c["tags_format"]]
	}

	// Global tags
	// They are sent with each metric
	// It can various things, such as node name, datacenter, ...
	if tags, ok := c["tags"]; ok {
		for _, item := range strings.Split(tags, ",") {
			l := strings.Split(item, "=")

			// If it has the correct format (key=value)
			if len(l) == 2 {
				t := tag{
					K: l[0],
					V: config.ValueOf(l[1]),
				}

				conf.Client.Tags = append(conf.Client.Tags, t)
			}
		}
	}

	// Prefi format
	if c["tags_format"] != "" {
		conf.Conn.TagFormat = tagsFormats[c["tags_format"]]
	}

	return conf
}
