package statsd

import (
	"fmt"
	"strings"
	"time"

	statsd "gopkg.in/alexcesaro/statsd.v2"

	"github.com/stairlin/lego/config"
	"github.com/stairlin/lego/log"
	"github.com/stairlin/lego/stats"
)

const Name = "statsd"

var tagsFormats = map[string]statsd.TagFormat{
	"influxdb": statsd.InfluxDB,
	"datadog":  statsd.Datadog,
}

func New(c map[string]string) (stats.Stats, error) {
	var opts []statsd.Option

	// Address
	if _, ok := c["port"]; ok {
		addr := config.ValueOf(c["addr"])
		port := config.ValueOf(c["port"])

		opt := statsd.Address(fmt.Sprintf("%s:%s", addr, port))
		opts = append(opts, opt)
	}

	// Tags format
	if c["tags_format"] != "" {
		f := tagsFormats[c["tags_format"]]
		opts = append(opts, statsd.TagsFormat(f))
	}

	// Custom tags
	if tags, ok := c["tags"]; ok {
		for _, tag := range strings.Split(tags, ",") {
			opts = append(opts, statsd.Tags(config.ValueOf(tag)))
		}
	}

	client, err := statsd.New(opts...)
	if err != nil {
		// If nothing is listening on the target port, an error is returned and
		// the returned client does nothing but is still usable. So we can
		// just log the error and go on.
		return nil, err
	}

	return &Client{
		C: client,
	}, nil
}

type Client struct {
	C      *statsd.Client
	logger log.Logger
}

func (c *Client) Start() {
	c.logger.Tracef("statsd: connecting...")
}
func (c *Client) Stop() {
	c.C.Close()
}
func (c *Client) SetLogger(l log.Logger) {
	c.logger = l
}

func (c *Client) Count(key string, n interface{}, meta ...map[string]string) {
	c.C.Count(key, n)
}
func (c *Client) Inc(key string, meta ...map[string]string) {
	c.C.Count(key, 1)
}
func (c *Client) Dec(key string, meta ...map[string]string) {
	c.C.Count(key, -1)
}
func (c *Client) Gauge(key string, n interface{}, meta ...map[string]string) {
	c.C.Gauge(key, n)
}
func (c *Client) Timing(key string, t time.Duration, meta ...map[string]string) {
	d := t.Nanoseconds() / 1000000
	c.C.Timing(key, d)
}
