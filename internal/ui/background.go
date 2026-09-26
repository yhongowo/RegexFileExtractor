package ui

import (
	"context"

	"regexfileextractor/internal/config"
	"regexfileextractor/internal/core"
)

// Files are immutable after scanning; only the compact selection bitmap is copied.
const asyncPlanThreshold = 1000

type planRequest struct {
	version      uint64
	files        []core.File
	selected     []bool
	layout       core.Layout
	scratch      []core.File
	outputBuffer map[string]string
}

func calculatePlan(ctx context.Context, request *planRequest) ([]core.Entry, map[string]string, error) {
	clear(request.scratch)
	selected := request.scratch[:0]
	if cap(selected) < len(request.files) {
		selected = make([]core.File, 0, len(request.files))
	}
	// Keep the entire used range so a later calculation clears stale references.
	defer func() { request.scratch = selected }()
	for i, file := range request.files {
		if i%1024 == 0 {
			if err := ctx.Err(); err != nil {
				return nil, nil, err
			}
		}
		if request.selected[i] {
			selected = append(selected, file)
		}
	}
	plan, err := core.PlanContext(ctx, selected, request.layout)
	if err != nil {
		return nil, nil, err
	}
	outputs := request.outputBuffer
	if outputs == nil {
		outputs = make(map[string]string, len(plan))
	} else {
		clear(outputs)
	}
	for i, entry := range plan {
		if i%1024 == 0 {
			if err := ctx.Err(); err != nil {
				return nil, nil, err
			}
		}
		outputs[entry.File.Path] = entry.Relative
	}
	return plan, outputs, nil
}

// Only one calculation runs at a time; rapid edits replace the pending request.
func (c *Controller) launchPlan() {
	if c.planRunning || c.nextPlan == nil || c.closed {
		return
	}
	request := *c.nextPlan
	request.scratch = c.planScratch
	c.planScratch = nil
	c.nextPlan = nil
	ctx, cancel := context.WithCancel(context.Background())
	c.planCancel = cancel
	c.planRunning = true
	go func() {
		plan, outputs, err := calculatePlan(ctx, &request)
		cancel()
		c.dispatch(func() {
			c.planRunning = false
			if !c.closed && len(c.files) > 0 {
				c.planScratch = request.scratch
			}
			c.planCancel = nil
			if c.closed {
				return
			}
			if request.version == c.planVersion {
				c.planPending = false
				c.applyPlan(plan, outputs, err)
				c.resultList.Refresh()
				c.updateControls()
			}
			c.launchPlan()
		})
	}()
}

func (c *Controller) applyPlan(plan []core.Entry, outputs map[string]string, err error) {
	c.plan, c.outputs = plan, outputs
	if c.status != nil && c.status.Text == c.tr("planning") {
		c.status.SetText(c.planStatus)
	}
	warning := ""
	if err != nil {
		warning = err.Error()
	}
	c.setPlanWarning(warning)
}

func (c *Controller) setPlanWarning(warning string) {
	if c.warning == nil {
		return
	}
	previous := c.warning.Text
	setLabel(c.warning, warning)
	if warning == "" {
		c.warning.Hide()
	} else {
		c.warning.Show()
	}
	if previous != warning && c.settings != nil {
		c.settings.invalidateMeasure()
		c.settings.Refresh()
	}
}

type saveRequest struct {
	cfg  config.Config
	done func(error)
}

// Writes are serialized, including rule edits and the final close flush.
// Adjacent ordinary preference saves coalesce; callbacks are never discarded.
func (c *Controller) saveConfig(cfg config.Config, done func(error)) {
	cfg.Rules = append([]core.Rule(nil), cfg.Rules...)
	request := saveRequest{cfg, done}
	n := len(c.saveQueue)
	if done == nil && n > 0 && c.saveQueue[n-1].done == nil {
		c.saveQueue[n-1] = request
	} else {
		c.saveQueue = append(c.saveQueue, request)
	}
	c.launchSave()
}

func (c *Controller) launchSave() {
	if c.saveRunning || len(c.saveQueue) == 0 || c.closed {
		return
	}
	request := c.saveQueue[0]
	c.saveQueue[0] = saveRequest{}
	c.saveQueue = c.saveQueue[1:]
	c.saveRunning = true
	path := c.configPath
	go func() {
		err := config.Save(path, request.cfg)
		c.dispatch(func() {
			c.saveRunning = false
			if c.closed {
				return
			}
			if request.done != nil {
				request.done(err)
			} else if err != nil && !c.closing {
				c.fail(err)
			}
			c.launchSave()
		})
	}()
}
