package config

import (
	"fmt"
	"strings"
	"text/template"

	"go.uber.org/zap"
)

type TemplateRenderer struct {
	BaseTemplate *template.Template
	Logger       *zap.SugaredLogger
	Data         interface{}
}

func (r *TemplateRenderer) RenderValueOrDie(text string) string {
	v, err := r.RenderValue(text)
	if err != nil {
		r.Logger.Fatal(err)
	}
	return v
}

func (r *TemplateRenderer) RenderValue(text string) (string, error) {
	t, err := r.BaseTemplate.Clone()
	if err != nil {
		return "", fmt.Errorf("Unable to clone template: %w", err)
	}

	t, err = t.New("").Parse(text)
	if err != nil {
		return "", fmt.Errorf("Unable to parse template: %w", err)
	}

	var buf strings.Builder
	err = t.Execute(&buf, r.Data)
	if err != nil {
		return "", fmt.Errorf("Unable to render template: %w", err)
	}

	return buf.String(), nil
}

func (r *TemplateRenderer) RenderCommandOrDie(text string, cmd []string, flagName string) []string {
	v := r.RenderValueOrDie(text)
	if v == "" {
		return cmd
	}

	if flagName == "" {
		return append(cmd, v)
	} else {
		return append(cmd, fmt.Sprintf("--%s", flagName), v)
	}
}
