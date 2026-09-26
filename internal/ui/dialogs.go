package ui

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"strings"

	"regexfileextractor/internal/config"
	"regexfileextractor/internal/core"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

func (c *Controller) browse(entry *widget.Entry) {
	if showNativeFolderPicker(c.Window, entry.Text, func(path string, err error) {
		if err != nil {
			c.fail(err)
		} else if path != "" {
			entry.SetText(path)
		}
	}) {
		return
	}
	d := dialog.NewFolderOpen(func(uri fyne.ListableURI, err error) {
		if err != nil {
			c.fail(err)
			return
		}
		if uri != nil {
			entry.SetText(uri.Path())
		}
	}, c.Window)
	d.SetConfirmText(c.tr("ok"))
	d.SetDismissText(c.tr("cancel"))
	if info, err := os.Stat(entry.Text); err == nil && info.IsDir() {
		if uri, err := storage.ListerForURI(storage.NewFileURI(entry.Text)); err == nil {
			d.SetLocation(uri)
		}
	}
	d.Show()
	// FileDialog creates its internal popup in Show. Resizing it earlier makes
	// Fyne dereference the not-yet-created popup and crashes the application.
	d.Resize(fyne.NewSize(800, 520))
}

func (c *Controller) editRule(edit bool) {
	original := core.Rule{}
	if edit {
		var ok bool
		original, ok = c.currentRule()
		if !ok {
			c.fail(fmt.Errorf("%s", c.tr("chooseRule")))
			return
		}
	}
	name, pattern, sample := widget.NewEntry(), widget.NewEntry(), widget.NewEntry()
	pattern.TextStyle.Monospace = true
	sample.TextStyle.Monospace = true
	name.SetText(original.Name)
	pattern.SetText(original.Pattern)
	pattern.SetPlaceHolder(`X\d+Y\d+\.csv`)
	sample.SetText("X01Y01.csv")
	validation := widget.NewLabel("")
	validation.Wrapping = fyne.TextWrapWord
	match := widget.NewLabel("")
	title := c.tr("newRule")
	if edit {
		title = c.tr("editRule")
	}
	var d *dialog.CustomDialog
	var save *widget.Button
	validate := func() error {
		if strings.TrimSpace(name.Text) == "" {
			return fmt.Errorf("%s", c.tr("nameRequired"))
		}
		for _, rule := range c.cfg.Rules {
			if rule.ID != original.ID && strings.EqualFold(strings.TrimSpace(name.Text), strings.TrimSpace(rule.Name)) {
				return fmt.Errorf("%s", c.tr("duplicateName"))
			}
		}
		if pattern.Text == "" {
			return fmt.Errorf("%s", c.tr("patternRequired"))
		}
		if _, err := (core.Rule{Name: name.Text, Pattern: pattern.Text}).Compile(); err != nil {
			return fmt.Errorf(c.tr("invalidPattern"), err.Error())
		}
		return nil
	}
	save = widget.NewButton(c.tr("save"), func() {
		if err := validate(); err != nil {
			validation.SetText(err.Error())
			return
		}
		rule := core.Rule{ID: original.ID, Name: strings.TrimSpace(name.Text), Pattern: pattern.Text}
		if !edit {
			var id [16]byte
			if _, err := rand.Read(id[:]); err != nil {
				c.fail(err)
				return
			}
			rule.ID = hex.EncodeToString(id[:])
		}
		next := c.cfg
		next.Rules = append([]core.Rule(nil), c.cfg.Rules...)
		if edit {
			for i := range next.Rules {
				if next.Rules[i].ID == rule.ID {
					next.Rules[i] = rule
					break
				}
			}
		} else {
			next.Rules = append(next.Rules, rule)
		}
		next.SelectedRule = rule.ID
		if c.loadErr != nil {
			c.fail(fmt.Errorf("%s", c.tr("readOnly")))
			return
		}
		c.cfg = next
		c.persist()
		c.invalidate()
		d.Hide()
		c.build()
	})
	save.Importance = widget.HighImportance
	update := func(string) {
		if err := validate(); err != nil {
			validation.SetText(err.Error())
			save.Disable()
			match.SetText("")
			return
		}
		validation.SetText("")
		save.Enable()
		re, _ := (core.Rule{Name: name.Text, Pattern: pattern.Text}).Compile()
		if re.MatchString(sample.Text) {
			match.SetText(c.tr("match"))
		} else {
			match.SetText(c.tr("noMatch"))
		}
	}
	name.OnChanged = update
	pattern.OnChanged = update
	sample.OnChanged = update
	hint := widget.NewLabel(c.tr("patternHint"))
	hint.Wrapping = fyne.TextWrapWord
	form := widget.NewForm(widget.NewFormItem(c.tr("name"), name), widget.NewFormItem(c.tr("pattern"), pattern), widget.NewFormItem(c.tr("sample"), sample))
	cancel := widget.NewButton(c.tr("cancel"), func() { d.Hide() })
	d = dialog.NewCustomWithoutButtons(title, container.NewVBox(form, hint, match, validation, container.NewHBox(cancel, save)), c.Window)
	d.Resize(fyne.NewSize(660, 350))
	update("")
	d.Show()
	c.Window.Canvas().Focus(name)
}

func (c *Controller) deleteRule() {
	rule, ok := c.currentRule()
	if !ok {
		c.fail(fmt.Errorf("%s", c.tr("chooseRule")))
		return
	}
	c.confirm(c.tr("delete"), fmt.Sprintf(c.tr("deleteRule"), rule.Name), c.tr("delete"), func(ok bool) {
		if !ok {
			return
		}
		next := c.cfg
		next.Rules = nil
		for _, r := range c.cfg.Rules {
			if r.ID != rule.ID {
				next.Rules = append(next.Rules, r)
			}
		}
		next.SelectedRule = ""
		if len(next.Rules) > 0 {
			next.SelectedRule = next.Rules[0].ID
		}
		if c.loadErr != nil {
			c.fail(fmt.Errorf("%s", c.tr("readOnly")))
			return
		}
		c.cfg = next
		c.persist()
		c.invalidate()
		c.build()
	})
}

func (c *Controller) showText(title, text string) {
	view := widget.NewLabel(text)
	view.Wrapping = fyne.TextWrapWord
	copy := widget.NewButtonWithIcon(c.tr("copyText"), theme.ContentCopyIcon(), func() { c.Window.Clipboard().SetContent(text) })
	d := dialog.NewCustom(title, c.tr("ok"), container.NewBorder(nil, copy, nil, nil, container.NewVScroll(view)), c.Window)
	d.Resize(fyne.NewSize(300, 240))
	d.Show()
}
func (c *Controller) about() {
	if c.loadErr != nil {
		c.ShowLoadError()
		return
	}
	content := widget.NewLabel(fmt.Sprintf(c.tr("aboutText"), c.configPath))
	content.Wrapping = fyne.TextWrapWord
	d := dialog.NewCustom(c.tr("about"), c.tr("ok"), content, c.Window)
	d.Resize(fyne.NewSize(300, 200))
	d.Show()
}

func (c *Controller) ShowLoadError() {
	if c.loadErr == nil {
		return
	}
	c.confirm(c.tr("configError"), fmt.Sprintf(c.tr("configRecovery"), c.loadErr.Error()), c.tr("recover"), func(ok bool) {
		if !ok {
			return
		}
		backup, err := config.Recover(c.configPath)
		if err != nil {
			c.fail(err)
			return
		}
		c.loadErr = nil
		c.cfg = config.Default()
		c.build()
		c.showText(c.tr("configError"), backup)
	})
}
