package certificate

import (
	"fmt"
	"time"
)

type WebhookType string

const (
	MutatingWebhook   WebhookType = "Mutating"
	ValidatingWebhook WebhookType = "Validating"
	OneYearDuration               = 365 * 24 * time.Hour
)

type Options struct {

	// webhookName The Mutating or Validating Webhook configuration name
	WebhookName string

	// webhookType The Mutating or Validating Webhook configuration type
	WebhookType WebhookType

	// The namespace where ca secret will be created or service secrets
	// for ClientConfig that has URL instead of ServiceRef
	Namespace string

	// CARotateInterval configurated duration for CA and certificate
	CARotateInterval time.Duration

	// CAOverlapInterval the duration of CA Certificates at CABundle if
	// not set it will default to CARotateInterval
	CAOverlapInterval time.Duration

	// CertRotateInterval configurated duration for of service certificate
	// the the webhook configuration is referencing different services all
	// of them will share the same duration
	CertRotateInterval time.Duration

	// CertOverlapInterval the duration of service certificates at bundle if
	// not set it will default to CertRotateInterval
	CertOverlapInterval time.Duration

	// ExtraLabels extra labels that will be added to created secrets
	ExtraLabels map[string]string
}

func (o *Options) validate() error {
	if o.WebhookName == "" {
		return fmt.Errorf("failed validating certificate options, 'WebhookName' field is missing")
	}
	if o.Namespace == "" {
		return fmt.Errorf("failed validating certificate options, 'Namespace' field is missing")
	}

	if o.CAOverlapInterval > o.CARotateInterval {
		return fmt.Errorf("failed validating certificate options, 'CAOverlapInterval' has to be <= 'CARotateInterval'")
	}

	if o.CertRotateInterval > o.CARotateInterval {
		return fmt.Errorf("failed validating certificate options, 'CertRotateInterval' has to be <= 'CARotateInterval'")
	}

	if o.CertOverlapInterval > o.CertRotateInterval {
		return fmt.Errorf("failed validating certificate options, 'CertOverlapInterval' has to be <= 'CertRotateInterval'")
	}

	if o.WebhookType != MutatingWebhook && o.WebhookType != ValidatingWebhook {
		return fmt.Errorf("failed validating certificate options, 'WebhookType' has to be %s or %s", MutatingWebhook, ValidatingWebhook)
	}

	return nil

}

func (o Options) withDefaults() Options {
	withDefaultsOptions := o
	if o.WebhookType == "" {
		withDefaultsOptions.WebhookType = MutatingWebhook
	}

	if o.CARotateInterval == 0 {
		withDefaultsOptions.CARotateInterval = OneYearDuration
	}

	if o.CAOverlapInterval == 0 {
		withDefaultsOptions.CAOverlapInterval = withDefaultsOptions.CARotateInterval
	}

	if o.CertRotateInterval == 0 {
		withDefaultsOptions.CertRotateInterval = withDefaultsOptions.CARotateInterval
	}

	if o.CertOverlapInterval == 0 {
		withDefaultsOptions.CertOverlapInterval = withDefaultsOptions.CertRotateInterval
	}
	return withDefaultsOptions
}

func (o *Options) setDefaultsAndValidate() error {
	withDefaultsOptions := o.withDefaults()
	err := withDefaultsOptions.validate()
	if err != nil {
		return err
	}
	*o = withDefaultsOptions
	return nil
}
