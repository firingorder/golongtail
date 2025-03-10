package commands

import (
	"time"

	"github.com/DanEngelbrecht/golongtail/longtaillib"
	"github.com/DanEngelbrecht/golongtail/longtailutils"
	"github.com/DanEngelbrecht/golongtail/remotestore"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func initRemoteStore(
	numWorkerCount int,
	blobStoreURI string,
	hashAlgorithm string) ([]longtailutils.StoreStat, []longtailutils.TimeStat, error) {
	const fname = "initRemoteStore"
	log := logrus.WithFields(logrus.Fields{
		"fname":          fname,
		"numWorkerCount": numWorkerCount,
		"blobStoreURI":   blobStoreURI,
		"hashAlgorithm":  hashAlgorithm,
	})
	log.Debug(fname)

	storeStats := []longtailutils.StoreStat{}
	timeStats := []longtailutils.TimeStat{}

	setupStartTime := time.Now()

	jobs := longtaillib.CreateBikeshedJobAPI(uint32(numWorkerCount), 0)
	defer jobs.Dispose()

	remoteIndexStore, err := remotestore.CreateBlockStoreForURI(blobStoreURI, "", jobs, numWorkerCount, 8388608, 1024, remotestore.Init)
	if err != nil {
		return storeStats, timeStats, errors.Wrap(err, fname)
	}
	defer remoteIndexStore.Dispose()
	setupTime := time.Since(setupStartTime)
	timeStats = append(timeStats, longtailutils.TimeStat{"Setup", setupTime})

	getExistingContentStartTime := time.Now()
	retargetStoreIndex, err := longtailutils.GetExistingStoreIndexSync(remoteIndexStore, []uint64{}, 0)
	if err != nil {
		err = errors.Wrapf(err, "Failed getting store index from store `%s`", blobStoreURI)
		return storeStats, timeStats, errors.Wrap(err, fname)
	}
	defer retargetStoreIndex.Dispose()
	getExistingContentTime := time.Since(getExistingContentStartTime)
	timeStats = append(timeStats, longtailutils.TimeStat{"Get store index", getExistingContentTime})

	flushStartTime := time.Now()

	err = longtailutils.FlushStoreSync(&remoteIndexStore)
	if err != nil {
		return storeStats, timeStats, errors.Wrap(err, fname)
	}

	flushTime := time.Since(flushStartTime)
	timeStats = append(timeStats, longtailutils.TimeStat{"Flush", flushTime})

	remoteStoreStats, err := remoteIndexStore.GetStats()
	if err == nil {
		storeStats = append(storeStats, longtailutils.StoreStat{"Remote", remoteStoreStats})
	}

	return storeStats, timeStats, nil
}

type InitRemoteStoreCmd struct {
	StorageURIOption
	HashingOption
}

func (r *InitRemoteStoreCmd) Run(ctx *Context) error {
	storeStats, timeStats, err := initRemoteStore(
		ctx.NumWorkerCount,
		r.StorageURI,
		r.Hashing)
	ctx.StoreStats = append(ctx.StoreStats, storeStats...)
	ctx.TimeStats = append(ctx.TimeStats, timeStats...)
	return err
}
