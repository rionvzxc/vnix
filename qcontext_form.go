package vnix

import (
	"net/http"
)

// BindForm binds form fields into a struct using form tags.
func (this *QContext) BindForm(value any) error {
	if err := this.parseForm(); err != nil {
		return err
	}
	return bindValues(value, "form", TValues(this.httpRequest.PostForm))
}

// Form returns the first form value for name or defaultValue.
func (this *QContext) Form(name string, defaultValue string) string {
	if err := this.parseForm(); err != nil {
		return defaultValue
	}
	return TValues(this.httpRequest.PostForm).GetDefault(name, defaultValue)
}

// FormValues returns a copy of all values for a form field.
func (this *QContext) FormValues(name string) []string {
	if err := this.parseForm(); err != nil {
		return nil
	}
	return append([]string(nil), this.httpRequest.PostForm[name]...)
}

// RequestForm returns a clone of all parsed form values, or nil if parsing fails.
func (this *QContext) RequestForm() TValues {
	if err := this.parseForm(); err != nil {
		return nil
	}
	return TValues(this.httpRequest.PostForm).Clone()
}

func (this *QContext) parseForm() error {
	if this.bodyFormParsed {
		return this.bodyFormErr
	}
	if this.bodyRead {
		return errBodyAlreadyRead
	}
	this.bodyRead = true
	this.bodyFormParsed = true
	this.bodyFormErr = this.httpRequest.ParseForm()

	if this.bodyFormErr != nil {
		return this.bodyFormErr
	}
	this.bodyFormErr = this.httpRequest.ParseMultipartForm(32 << 20)

	if this.bodyFormErr == http.ErrNotMultipart {
		this.bodyFormErr = nil
	}
	return this.bodyFormErr
}
