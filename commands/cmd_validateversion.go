package commands

import (
	"time"

	"github.com/DanEngelbrecht/golongtail/longtaillib"
	"github.com/DanEngelbrecht/golongtail/longtailutils"
	"github.com/DanEngelbrecht/golongtail/remotestore"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func validateVersion(
	numWorkerCount int,
	blobStoreURI string,
	versionIndexPath string) ([]longtailutils.StoreStat, []longtailutils.TimeStat, error) {
	const fname = "validateVersion"
	log := logrus.WithFields(logrus.Fields{
		"numWorkerCount":   numWorkerCount,
		"blobStoreURI":     blobStoreURI,
		"versionIndexPath": versionIndexPath,
	})
	log.Debug(fname)

	storeStats := []longtailutils.StoreStat{}
	timeStats := []longtailutils.TimeStat{}

	setupStartTime := time.Now()

	jobs := longtaillib.CreateBikeshedJobAPI(uint32(numWorkerCount), 0)
	defer jobs.Dispose()

	// MaxBlockSize and MaxChunksPerBlock are just temporary values until we get the remote index settings
	indexStore, err := remotestore.CreateBlockStoreForURI(blobStoreURI, "", jobs, numWorkerCount, 8388608, 1024, remotestore.ReadOnly)
	if err != nil {
		return storeStats, timeStats, errors.Wrap(err, fname)
	}
	defer indexStore.Dispose()
	setupTime := time.Since(setupStartTime)
	timeStats = append(timeStats, longtailutils.TimeStat{"Setup", setupTime})

	readSourceStartTime := time.Now()
	vbuffer, err := longtailutils.ReadFromURI(versionIndexPath)
	if err != nil {
		return storeStats, timeStats, errors.Wrap(err, fname)
	}
	versionIndex, err := longtaillib.ReadVersionIndexFromBuffer(vbuffer)
	if err != nil {
		err = errors.Wrapf(err, "Cant parse version index from `%s`", versionIndexPath)
		return storeStats, timeStats, errors.Wrap(err, fname)
	}
	defer versionIndex.Dispose()
	readSourceTime := time.Since(readSourceStartTime)
	timeStats = append(timeStats, longtailutils.TimeStat{"Read source index", readSourceTime})

	getExistingContentStartTime := time.Now()
	remoteStoreIndex, err := longtailutils.GetExistingStoreIndexSync(indexStore, versionIndex.GetChunkHashes(), 0)
	if err != nil {
		return storeStats, timeStats, errors.Wrap(err, fname)
	}
	defer remoteStoreIndex.Dispose()
	getExistingContentTime := time.Since(getExistingContentStartTime)
	timeStats = append(timeStats, longtailutils.TimeStat{"Get content index", getExistingContentTime})

	validateStartTime := time.Now()
	err = longtaillib.ValidateStore(remoteStoreIndex, versionIndex)
	if err != nil {
		err = errors.Wrapf(err, "Validate failed for version index `%s`", versionIndexPath)
		return storeStats, timeStats, errors.Wrap(err, fname)
	}
	validateTime := time.Since(validateStartTime)
	timeStats = append(timeStats, longtailutils.TimeStat{"Validate", validateTime})

	return storeStats, timeStats, nil
}

type ValidateVersionCmd struct {
	StorageURIOption
	VersionIndexPathOption
}

func (r *ValidateVersionCmd) Run(ctx *Context) error {
	storeStats, timeStats, err := validateVersion(
		ctx.NumWorkerCount,
		r.StorageURI,
		r.VersionIndexPath)
	ctx.StoreStats = append(ctx.StoreStats, storeStats...)
	ctx.TimeStats = append(ctx.TimeStats, timeStats...)
	return err
}
