package ui

import (
	"context"
	"errors"
	"fmt"
	"image/color"
	"strings"

	"regexfileextractor/internal/config"
	"regexfileextractor/internal/core"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

type Controller struct {
	app                                    fyne.App
	Window                                 fyne.Window
	cfg                                    config.Config
	configPath                             string
	loadErr                                error
	files                                  []core.File
	selected                               []bool
	planScratch                            []core.File
	plan                                   []core.Entry
	outputs                                map[string]string
	busy                                   bool
	cancel                                 context.CancelFunc
	closeAfterCancel                       bool
	details                                string
	source, target                         *widget.Entry
	rule                                   *widget.Select
	pattern, status, selection, warning    *widget.Label
	emptyView                              fyne.CanvasObject
	resultList                             *widget.List
	resultsCard                            fyne.CanvasObject
	selectAllCheck                         *selectionHeaderCheck
	settings                               *settingsView
	languageButton                         *labeledIconButton
	progress                               *widget.ProgressBar
	activity                               *widget.ProgressBarInfinite
	scan, copy, cancelButton, detailButton *widget.Button
	controls                               []fyne.Disableable
}

func New(a fyne.App, cfg config.Config, path string, loadErr error) *Controller {
	c := &Controller{app: a, cfg: cfg, configPath: path, loadErr: loadErr, outputs: map[string]string{}}
	c.Window = a.NewWindow("Regex File Extractor")
	c.Window.SetCloseIntercept(c.close)
	c.build()
	c.Window.Resize(fyne.NewSize(800, 600))
	return c
}

func (c *Controller) build() {
	c.controls = nil
	c.languageButton = newLabeledIconButton(c.tr("switchLanguage"), languageIcon, func() {
		if c.cfg.Language == "zh" {
			c.setLanguage("en")
		} else {
			c.setLanguage("zh")
		}
	})
	about := newLabeledIconButton(c.tr("about"), infoIcon, c.about)
	workspaceTitle := canvas.NewText(c.tr("workspace"), color.NRGBA{R: 75, G: 85, B: 99, A: 255})
	workspaceTitle.TextSize = 12
	workspaceTitle.TextStyle = fyne.TextStyle{Bold: true}
	header := container.NewVBox(
		container.NewBorder(nil, nil, container.NewPadded(container.NewCenter(container.NewHBox(widget.NewIcon(fileIcon), workspaceTitle))), gapRow(8,
			outlineIconButton(c.languageButton),
			outlineIconButton(about),
		), nil),
		widget.NewSeparator(),
	)
	c.controls = append(c.controls, c.languageButton)

	c.source = widget.NewEntry()
	c.source.SetPlaceHolder(c.tr("sourceHint"))
	c.source.SetText(c.cfg.Source)
	c.source.OnChanged = func(value string) { c.cfg.Source = value; c.invalidate() }
	c.target = widget.NewEntry()
	c.target.SetPlaceHolder(c.tr("targetHint"))
	c.target.SetText(c.cfg.Destination)
	c.target.OnChanged = func(value string) { c.cfg.Destination = value; c.invalidate() }
	browseSource := newLabeledIconButton(c.tr("source")+" · "+c.tr("browse"), folderIcon, func() { c.browse(c.source) })
	browseTarget := newLabeledIconButton(c.tr("target")+" · "+c.tr("browse"), folderIcon, func() { c.browse(c.target) })
	c.rule = widget.NewSelect(c.ruleNames(), nil)
	if rule, ok := c.currentRule(); ok {
		c.rule.SetSelected(rule.Name)
	}
	c.rule.OnChanged = func(name string) {
		for _, rule := range c.cfg.Rules {
			if rule.Name == name {
				c.cfg.SelectedRule = rule.ID
				break
			}
		}
		c.refreshPattern()
		c.invalidate()
		c.persist()
	}
	c.pattern = widget.NewLabel("")
	c.pattern.TextStyle.Monospace = true
	c.pattern.Truncation = fyne.TextTruncateEllipsis
	c.refreshPattern()
	copyPattern := newLabeledIconButton(c.tr("copyPattern"), copyIcon, func() {
		c.Window.Clipboard().SetContent(c.pattern.Text)
	})
	add := newLabeledIconButton(c.tr("newRule"), addIcon, func() { c.editRule(false) })
	edit := newLabeledIconButton(c.tr("editRule"), editIcon, func() { c.editRule(true) })
	remove := newLabeledIconButton(c.tr("deleteRuleAction"), deleteIcon, c.deleteRule)
	left := settingsPanel(c.tr("inputs"), settingsIcon, container.NewVBox(
		fieldRow(c.tr("source"), container.NewBorder(nil, nil, nil, outlineIconButton(browseSource), c.source)),
		fieldRow(c.tr("target"), container.NewBorder(nil, nil, nil, outlineIconButton(browseTarget), c.target)),
		fieldRow(c.tr("rule"), container.NewBorder(nil, nil, nil, container.NewHBox(outlineIconButton(add), outlineIconButton(edit), outlineIconButton(remove)), c.rule)),
		widget.NewLabelWithStyle(c.tr("pattern"), fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		cardSurface(container.NewPadded(container.NewBorder(nil, nil, nil, outlineIconButton(copyPattern), c.pattern))),
	))

	layoutSelect := widget.NewRadioGroup([]string{"auto", "group", "flat"}, nil)
	layoutSelect.Horizontal = true
	layoutSelect.Required = true
	layoutSelect.SetSelected(string(c.cfg.Layout))
	layoutHint := widget.NewLabel(c.tr(string(c.cfg.Layout) + "Hint"))
	layoutHint.Wrapping = fyne.TextWrapWord
	layoutSelect.OnChanged = func(value string) {
		c.cfg.Layout = core.Layout(value)
		layoutHint.SetText(c.tr(value + "Hint"))
		if c.settings != nil {
			c.settings.invalidateMeasure()
			c.settings.Refresh()
		}
		c.refreshPlan()
		c.persist()
	}
	conflict := widget.NewSelect([]string{c.tr("skip"), c.tr("overwrite")}, nil)
	conflict.SetSelected(c.tr(string(c.cfg.Conflict)))
	conflict.OnChanged = func(value string) {
		c.cfg.Conflict = core.Skip
		if value == c.tr("overwrite") {
			c.cfg.Conflict = core.Overwrite
		}
		c.persist()
	}
	c.warning = widget.NewLabel("")
	c.warning.Wrapping = fyne.TextWrapWord
	right := settingsPanel(c.tr("outputs"), extractionIcon, container.NewVBox(
		widget.NewLabelWithStyle(c.tr("layout"), fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		layoutSelect, layoutHint, widget.NewSeparator(),
		fieldRow(c.tr("conflict"), conflict), c.warning,
	))
	c.controls = append(c.controls, c.source, c.target, browseSource, browseTarget, c.rule, add, edit, remove, copyPattern, layoutSelect, conflict)

	c.selection = widget.NewLabel("")
	resultsTitle := container.NewHBox(sectionHeading(c.tr("results"), listIcon), c.selection)
	resultsHeader := container.NewPadded(resultsTitle)
	var columnHeaders fyne.CanvasObject
	c.resultList, columnHeaders = c.makeResults()
	emptyTitle := canvas.NewText(c.tr("empty"), color.NRGBA{R: 31, G: 41, B: 55, A: 255})
	emptyTitle.TextSize = 14
	emptyTitle.TextStyle = fyne.TextStyle{Bold: true}
	emptyTitle.Alignment = fyne.TextAlignCenter
	emptyHint := canvas.NewText(c.tr("emptyHint"), color.NRGBA{R: 107, G: 114, B: 128, A: 255})
	emptyHint.TextSize = 12
	emptyHint.Alignment = fyne.TextAlignCenter
	emptyIcon := canvas.NewImageFromResource(mutedFileIcon)
	emptyIcon.FillMode = canvas.ImageFillContain
	emptyIcon.SetMinSize(fyne.NewSquareSize(32))
	c.emptyView = container.NewCenter(container.New(emptyStateLayout{}, emptyIcon, emptyTitle, emptyHint))
	resultBody := container.NewStack(c.resultList, c.emptyView)
	tableHeaderBackground := canvas.NewRectangle(color.NRGBA{R: 244, G: 246, B: 249, A: 255})
	results := cardSurface(container.NewBorder(
		container.NewVBox(resultsHeader, widget.NewSeparator(), container.NewStack(tableHeaderBackground, columnHeaders), widget.NewSeparator()),
		nil, nil, nil, resultBody,
	))
	c.resultsCard = results

	c.status = widget.NewLabel("")
	c.status.Truncation = fyne.TextTruncateEllipsis
	c.progress = widget.NewProgressBar()
	c.progress.Hide()
	c.activity = widget.NewProgressBarInfinite()
	c.activity.Stop()
	c.activity.Hide()
	c.scan = widget.NewButtonWithIcon(c.tr("scan"), searchIcon, c.startScan)
	c.scan.Importance = widget.HighImportance
	c.copy = widget.NewButtonWithIcon(c.tr("copy"), copyIcon, c.confirmCopy)
	c.cancelButton = widget.NewButton(c.tr("cancel"), c.stop)
	c.detailButton = widget.NewButtonWithIcon(c.tr("details"), fileIcon, func() { c.showText(c.tr("report"), c.details) })
	footActions := container.NewBorder(nil, nil, sizedButton(outlineButton(c.detailButton), 112, 34),
		gapRow(8,
			sizedButton(outlineButton(c.cancelButton), 76, 34),
			sizedButton(c.scan, 88, 34),
			sizedButton(outlineButton(c.copy), 128, 34),
		), c.status)
	foot := container.NewVBox(container.NewStack(c.progress, c.activity), widget.NewSeparator(), footActions)
	c.settings = newSettingsView(left, right)
	body := container.New(workspaceLayout{gap: 8}, c.settings, results)
	c.Window.SetContent(container.NewPadded(container.NewBorder(header, foot, nil, nil, body)))
	c.refreshPlan()
}

func (c *Controller) setLanguage(value string) {
	if c.cfg.Language == value {
		return
	}
	before := c.cfg.Language
	c.cfg.Language = value
	if !c.persist() {
		c.cfg.Language = before
		return
	}
	c.build()
}

func (c *Controller) currentRule() (core.Rule, bool) {
	for _, rule := range c.cfg.Rules {
		if rule.ID == c.cfg.SelectedRule {
			return rule, true
		}
	}
	return core.Rule{}, false
}
func (c *Controller) ruleNames() []string {
	names := make([]string, len(c.cfg.Rules))
	for i, r := range c.cfg.Rules {
		names[i] = r.Name
	}
	return names
}
func (c *Controller) refreshPattern() {
	if c.pattern == nil {
		return
	}
	if r, ok := c.currentRule(); ok {
		c.pattern.SetText(r.Pattern)
	} else {
		c.pattern.SetText("")
	}
}

func (c *Controller) invalidate() {
	c.files = nil
	c.selected = nil
	c.planScratch = nil
	c.outputs = make(map[string]string)
	c.details = ""
	if c.status != nil {
		c.status.SetText(c.tr("stale"))
	}
	if c.resultList != nil {
		c.refreshPlan()
	}
}
func (c *Controller) selectAll(value bool) {
	if c.busy {
		return
	}
	for i := range c.selected {
		c.selected[i] = value
	}
	c.refreshPlan()
}
func (c *Controller) refreshPlan() {
	clear(c.planScratch)
	selected := c.planScratch[:0]
	if cap(selected) < len(c.files) {
		selected = make([]core.File, 0, len(c.files))
	}
	var size int64
	for i, f := range c.files {
		if c.selected[i] {
			selected = append(selected, f)
			size += f.Size
		}
	}
	plan, err := core.Plan(selected, c.cfg.Layout)
	c.planScratch = selected
	c.plan = plan
	clear(c.outputs)
	for _, entry := range plan {
		c.outputs[entry.File.Path] = entry.Relative
	}
	if c.selection != nil {
		setLabel(c.selection, fmt.Sprintf(c.tr("selection"), len(selected), len(c.files), formatSize(size)))
	}
	if c.selectAllCheck != nil {
		allSelected := len(c.files) > 0 && len(selected) == len(c.files)
		c.selectAllCheck.label = c.tr("all")
		if allSelected {
			c.selectAllCheck.label = c.tr("none")
		}
		if c.selectAllCheck.Checked != allSelected {
			changed := c.selectAllCheck.OnChanged
			c.selectAllCheck.OnChanged = nil
			c.selectAllCheck.SetChecked(allSelected)
			c.selectAllCheck.OnChanged = changed
		}
	}
	if c.warning != nil {
		previousWarning := c.warning.Text
		warning := ""
		if err != nil {
			warning = err.Error()
		}
		setLabel(c.warning, warning)
		if c.warning.Text == "" {
			c.warning.Hide()
		} else {
			c.warning.Show()
		}
		if previousWarning != warning && c.settings != nil {
			c.settings.invalidateMeasure()
			c.settings.Refresh()
		}
	}
	if c.resultList != nil {
		c.resultList.Refresh()
	}
	if c.emptyView != nil {
		if len(c.files) == 0 {
			c.emptyView.Show()
		} else {
			c.emptyView.Hide()
		}
	}
	c.updateControls()
}
func (c *Controller) updateControls() {
	for _, control := range c.controls {
		if c.busy {
			control.Disable()
		} else {
			control.Enable()
		}
	}
	if c.selectAllCheck != nil {
		setDisabled(c.selectAllCheck, c.busy || len(c.files) == 0)
	}
	if c.scan == nil {
		return
	}
	if c.busy {
		c.scan.Disable()
		c.copy.Disable()
		c.cancelButton.Enable()
	} else {
		c.scan.Enable()
		c.cancelButton.Disable()
		if len(c.plan) > 0 {
			c.copy.Enable()
		} else {
			c.copy.Disable()
		}
	}
	if c.details == "" {
		c.detailButton.Disable()
	} else {
		c.detailButton.Enable()
	}
	// Static outline SVGs retain their strokes; update only when a state flips.
	setButtonIcon(c.scan, searchIcon, mutedSearchIcon)
	setButtonIcon(c.copy, copyIcon, mutedCopyIcon)
	setButtonIcon(c.detailButton, fileIcon, mutedFileIcon)
}

func setButtonIcon(button *widget.Button, enabled, disabled fyne.Resource) {
	want := enabled
	if button.Disabled() {
		want = disabled
	}
	if button.Icon != want {
		button.SetIcon(want)
	}
}
func (c *Controller) persist() bool {
	if c.loadErr != nil {
		c.fail(errors.New(c.tr("readOnly")))
		return false
	}
	if err := config.Save(c.configPath, c.cfg); err != nil {
		c.fail(err)
		return false
	}
	return true
}

func (c *Controller) startScan() {
	if c.busy {
		return
	}
	rule, ok := c.currentRule()
	if !ok {
		c.fail(errors.New(c.tr("chooseRule")))
		return
	}
	if strings.TrimSpace(c.cfg.Source) == "" {
		c.fail(errors.New(c.tr("chooseSource")))
		return
	}
	if !c.persist() {
		return
	}
	c.invalidate()
	c.status.SetText(fmt.Sprintf(c.tr("scanProgress"), 0, 0))
	ctx := c.begin(true)
	opts := core.ScanOptions{Source: c.cfg.Source, Exclude: c.cfg.Destination, Rule: rule}
	go func() {
		result, err := core.Scan(ctx, opts, func(p core.ScanProgress) {
			// Scan already throttles progress updates. Do not make filesystem work
			// wait for the next GUI frame; that causes visible stalls under load.
			fyne.Do(func() { setLabel(c.status, fmt.Sprintf(c.tr("scanProgress"), p.Visited, p.Matched)) })
		})
		fyne.Do(func() {
			if c.finish() {
				return
			}
			if errors.Is(err, context.Canceled) {
				c.status.SetText(c.tr("scanCancelled"))
				return
			}
			if err != nil {
				c.status.SetText(c.tr("error"))
				c.fail(err)
				return
			}
			c.files = result.Files
			c.selected = make([]bool, len(c.files))
			for i := range c.selected {
				c.selected[i] = true
			}
			c.status.SetText(fmt.Sprintf(c.tr("scanDone"), result.Visited, len(result.Files), result.WarningCount))
			if result.WarningCount > 0 {
				c.details = c.status.Text + "\n\n" + c.tr("reportLimit") + "\n" + strings.Join(result.Warnings, "\n")
			}
			c.refreshPlan()
		})
	}()
}
func (c *Controller) confirmCopy() {
	if c.busy || len(c.plan) == 0 {
		return
	}
	if strings.TrimSpace(c.cfg.Destination) == "" {
		c.fail(errors.New(c.tr("chooseTarget")))
		return
	}
	if c.cfg.Conflict == core.Overwrite {
		c.confirm(c.tr("overwriteTitle"), fmt.Sprintf(c.tr("overwriteBody"), len(c.plan), c.cfg.Destination), c.tr("copy"), func(ok bool) {
			if ok {
				c.startCopy()
			}
		})
	} else {
		c.startCopy()
	}
}
func (c *Controller) startCopy() {
	if c.busy || len(c.plan) == 0 {
		return
	}
	if !c.persist() {
		return
	}
	entries := c.plan // Immutable while the copy task locks selection and settings.
	target, conflict := c.cfg.Destination, c.cfg.Conflict
	c.details = ""
	ctx := c.begin(false)
	c.status.SetText(fmt.Sprintf(c.tr("copyProgress"), 0, len(entries), 0, 0, 0))
	go func() {
		result, err := core.Copy(ctx, target, entries, conflict, func(p core.CopyProgress) {
			// Copy progress is throttled by core.Copy. Queue it for the UI instead
			// of blocking the copy goroutine behind rendering.
			fyne.Do(func() {
				c.progress.SetValue(float64(p.Done) / float64(p.Total))
				c.status.SetText(fmt.Sprintf(c.tr("copyProgress"), p.Done, p.Total, p.Copied, p.Skipped, p.Failed))
			})
		})
		fyne.Do(func() {
			if c.finish() {
				return
			}
			key := "copyDone"
			if errors.Is(err, context.Canceled) {
				key = "copyCancelled"
			} else if err != nil {
				key = "copyFailed"
				c.fail(err)
			}
			c.status.SetText(fmt.Sprintf(c.tr(key), result.Copied, result.Skipped, result.Failed))
			c.details = c.status.Text
			if len(result.Errors) > 0 {
				c.details += "\n\n" + c.tr("reportLimit") + "\n" + strings.Join(result.Errors, "\n")
			}
			if err != nil {
				c.details += "\n" + err.Error()
			}
			c.updateControls()
			if result.Failed > 0 {
				c.showText(c.tr("report"), c.details)
			}
		})
	}()
}
func (c *Controller) begin(scanning bool) context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	c.cancel = cancel
	c.busy = true
	c.progress.SetValue(0)
	if scanning {
		c.progress.Hide()
		c.activity.Show()
		c.activity.Start()
	} else {
		c.activity.Stop()
		c.activity.Hide()
		c.progress.Show()
	}
	c.updateControls()
	c.resultList.Refresh()
	return ctx
}
func (c *Controller) finish() bool {
	if c.cancel != nil {
		c.cancel()
		c.cancel = nil
	}
	c.busy = false
	c.activity.Stop()
	c.activity.Hide()
	c.progress.Hide()
	c.updateControls()
	c.resultList.Refresh()
	if c.closeAfterCancel {
		c.Window.SetCloseIntercept(nil)
		c.Window.Close()
		return true
	}
	return false
}
func (c *Controller) stop() {
	if c.cancel != nil {
		c.cancel()
		c.cancelButton.Disable()
		c.status.SetText(c.tr("cancelling"))
	}
}
func (c *Controller) close() {
	if c.busy {
		c.confirm(c.tr("closeTitle"), c.tr("closeBody"), c.tr("ok"), func(ok bool) {
			if ok {
				c.closeAfterCancel = true
				c.stop()
			}
		})
		return
	}
	if c.loadErr == nil {
		if err := config.Save(c.configPath, c.cfg); err != nil {
			c.confirm(c.tr("error"), fmt.Sprintf(c.tr("exitWithoutSaving"), err.Error()), c.tr("ok"), func(ok bool) {
				if ok {
					c.Window.SetCloseIntercept(nil)
					c.Window.Close()
				}
			})
			return
		}
	}
	c.Window.SetCloseIntercept(nil)
	c.Window.Close()
}

func formatSize(size int64) string {
	if size < 1024 {
		return fmt.Sprintf("%d B", size)
	}
	value := float64(size)
	units := []string{"B", "KB", "MB", "GB", "TB"}
	unit := 0
	for value >= 1024 && unit < len(units)-1 {
		value /= 1024
		unit++
	}
	return fmt.Sprintf("%.1f %s", value, units[unit])
}

func (c *Controller) fail(err error) {
	d := dialog.NewInformation(c.tr("error"), err.Error(), c.Window)
	d.SetDismissText(c.tr("ok"))
	d.Show()
}
func (c *Controller) confirm(title, body, accept string, callback func(bool)) {
	d := dialog.NewConfirm(title, body, callback, c.Window)
	d.SetConfirmText(accept)
	d.SetDismissText(c.tr("cancel"))
	d.Show()
}
