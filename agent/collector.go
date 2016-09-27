package agent

import (
	"strings"
	"time"

	"github.com/startover/cloudinsight-agent/common/api"
	"github.com/startover/cloudinsight-agent/common/config"
	"github.com/startover/cloudinsight-agent/common/emitter"
	"github.com/startover/cloudinsight-agent/common/gohai"
	"github.com/startover/cloudinsight-agent/common/log"
)

const metadataUpdateInterval = 4 * time.Hour

// Collector contains the output configuration
type Collector struct {
	*emitter.Emitter

	api   *api.API
	conf  *config.Config
	start time.Time
}

// NewCollector XXX
func NewCollector(conf *config.Config) *Collector {
	emitter := emitter.NewEmitter("Collector")
	api := api.NewAPI(conf.GetForwarderAddrWithScheme(), conf.GlobalConfig.LicenseKey, 5*time.Second)

	c := &Collector{
		Emitter: emitter,
		api:     api,
		conf:    conf,
		start:   time.Now(),
	}
	c.Emitter.Parent = c

	return c
}

// Post XXX
func (c *Collector) Post(metrics []interface{}) error {
	start := time.Now()
	payload := NewPayload(c.conf)
	payload.Metrics = metrics

	if c.shouldSendMetadata() {
		log.Debug("We should send metadata.")

		payload.Gohai = gohai.GetMetadata()
		if c.conf.GlobalConfig.Tags != "" {
			hostTags := strings.Split(c.conf.GlobalConfig.Tags, ",")
			for i, tag := range hostTags {
				hostTags[i] = strings.TrimSpace(tag)
			}

			payload.HostTags = map[string]interface{}{
				"system": hostTags,
			}
		}
	}

	processes := gohai.GetProcesses()
	if c.IsFirstRun() {
		// When first run, we will retrieve processes to get cpuPercent.
		time.Sleep(1 * time.Second)
		processes = gohai.GetProcesses()
	}

	payload.Processes = map[string]interface{}{
		"processes":  processes,
		"licenseKey": c.conf.GlobalConfig.LicenseKey,
		"host":       c.conf.GetHostname(),
	}

	err := c.api.SubmitMetrics(payload)
	elapsed := time.Since(start)
	if err == nil {
		log.Infof("Write batch of %d metrics in %s\n",
			len(metrics), elapsed)
	}
	return err
}

func (c *Collector) shouldSendMetadata() bool {
	if c.IsFirstRun() {
		return true
	}

	if time.Since(c.start) >= metadataUpdateInterval {
		c.start = time.Now()
		return true
	}

	return false
}