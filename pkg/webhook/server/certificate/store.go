package certificate

import (
	"crypto/tls"
	"os"
	"path/filepath"

	"github.com/pkg/errors"

	"k8s.io/client-go/util/certificate"
)

type filePairStore struct {
	directory string
	certFile  string
	keyFile   string
	fileStore certificate.FileStore
}

// NewFilePairStore returns a concrete implementation of a Store that is based on
// client-go certificate.NewFileStore [1] it create two links for key and cert
// to the generated pem file client-go store.
//
// At "Update" it will do the following symlinks:
// ${directory}/${certFile} > ${directory}/tls-current.pem
// ${directory}/${keyFile} > ${directory}/tls-current.pem
// If rotation is enabled, future cert/key the two symlinks will be
// re-created on update.
//
// [1] https://godoc.org/k8s.io/client-go/util/certificate
func NewFilePairStore(directory string, certFile string, keyFile string) (*filePairStore, error) {
	keyPath := filepath.Join(directory, keyFile)
	certPath := filepath.Join(directory, certFile)
	s, err := certificate.NewFileStore(
		"tls",
		directory,
		directory,
		certPath,
		keyPath,
	)
	if err != nil {
		return nil, err
	}
	return &filePairStore{
		directory: directory,
		certFile:  certFile,
		keyFile:   keyFile,
		fileStore: s,
	}, nil
}

// CurrentPath returns the path to the current version of these certificates.
func (s filePairStore) CurrentPath() string {
	return s.fileStore.CurrentPath()
}

func (s filePairStore) Current() (*tls.Certificate, error) {
	return s.fileStore.Current()
}

// linkUpdatedCertTo link linkName to to "CurrentPath()" file if
// link is already there it will remove it and create it again.
func (s filePairStore) linkUpdatedCertTo(linkName string) error {
	updateCertPath := s.fileStore.CurrentPath()
	_, err := os.Lstat(linkName)
	if err != nil {
		if !os.IsNotExist(err) {
			return errors.Wrapf(err, "failed checking link %s to certFile %s", linkName, updateCertPath)
		}
	} else {
		// We have to remove the link to re-create it
		err = os.Remove(linkName)
		if err != nil {
			return errors.Wrapf(err, "failed removing link %s", linkName)
		}
	}
	// Create the 'updated' symlink pointing to the requested file name.
	err = os.Symlink(updateCertPath, linkName)
	if err != nil {
		return errors.Wrapf(err, "failed creating a link %s to the updated cert file %s", linkName, updateCertPath)
	}
	return nil
}

func (s filePairStore) keyFilePath() string {
	return filepath.Join(s.directory, s.keyFile)
}

func (s filePairStore) certFilePath() string {
	return filepath.Join(s.directory, s.certFile)
}

// FileExists checks if specified file exists.
func fileExists(filename string) (bool, error) {
	_, err := os.Stat(filename)
	if err != nil {
		if !os.IsNotExist(err) {
			return false, err
		} else {
			return false, nil
		}
	}
	return true, nil
}
func (s filePairStore) keyFileExists() (bool, error) {
	return fileExists(s.keyFilePath())
}
func (s filePairStore) certFileExists() (bool, error) {
	return fileExists(s.certFilePath())
}

// This will call clieng-go cert manager Update and link
// cert/key to the updated cert+key pem file
func (s filePairStore) Update(certData, keyData []byte) (*tls.Certificate, error) {
	cert, err := s.fileStore.Update(certData, keyData)
	if err != nil {
		return cert, errors.Wrap(err, "failed updating cer/key data")
	}

	err = s.linkUpdatedCertTo(s.certFilePath())
	if err != nil {
		return cert, errors.Wrapf(err, "failed linking cert to %s", s.certFilePath())
	}
	err = s.linkUpdatedCertTo(s.keyFilePath())
	if err != nil {
		return cert, errors.Wrapf(err, "failed linking key to %s", s.keyFilePath())
	}
	return cert, nil
}
