package sqlite

import (
	"github.com/sirupsen/logrus"

	"github.com/operator-framework/operator-registry/pkg/registry"
)

type SQLStrandedBundleRemover interface {
	Remove() error
}

// StrandedBundleRemover removes stranded bundles from the database
type StrandedBundleRemover struct {
	store registry.Load
}

var _ SQLStrandedBundleRemover = &StrandedBundleRemover{}

func NewSQLStrandedBundleRemover(store registry.Load) *StrandedBundleRemover {
	return &StrandedBundleRemover{
		store: store,
	}
}

func (d *StrandedBundleRemover) Remove() error {
	log := logrus.New()

	bundles, err := d.store.RemoveStrandedBundles()
	if err != nil {
		return err
	}

	if len(bundles) > 0 {
		log.Info("removing stranded bundles ", bundles)
	} else {
		log.Info("no stranded bundles found")
	}

	return nil
}
