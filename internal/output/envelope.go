package output

// Breadcrumb suggests a follow-up command to the consumer.
type Breadcrumb struct {
	Description string `json:"description"`
	Command     string `json:"command"`
}

// Envelope wraps output data with metadata for agent consumption.
type Envelope struct {
	OK          bool         `json:"ok"`
	Data        interface{}  `json:"data"`
	Summary     string       `json:"summary"`
	Breadcrumbs []Breadcrumb `json:"breadcrumbs"`
}

// ErrorEnvelope wraps an error for JSON consumers.
type ErrorEnvelope struct {
	OK    bool        `json:"ok"`
	Error string      `json:"error"`
	Data  interface{} `json:"data"`
}

// RenderEnvelope outputs data wrapped in an envelope with summary and breadcrumbs.
func (f *Formatter) RenderEnvelope(data interface{}, summary string, breadcrumbs []Breadcrumb) error {
	if breadcrumbs == nil {
		breadcrumbs = []Breadcrumb{}
	}
	return f.RenderJSON(Envelope{
		OK:          true,
		Data:        data,
		Summary:     summary,
		Breadcrumbs: breadcrumbs,
	})
}

// RenderError outputs an error wrapped in an envelope.
func (f *Formatter) RenderError(err error) error {
	return f.RenderJSON(ErrorEnvelope{
		OK:    false,
		Error: err.Error(),
		Data:  nil,
	})
}
