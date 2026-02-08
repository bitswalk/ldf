package stages

import (
	"github.com/bitswalk/ldf/src/ldfd/build"
	"github.com/bitswalk/ldf/src/ldfd/db"
	"github.com/bitswalk/ldf/src/ldfd/storage"
)

// DefaultStages returns the ordered default build pipeline stages.
// The caller should pass these to Manager.RegisterStages().
func DefaultStages(
	componentRepo *db.ComponentRepository,
	downloadJobRepo *db.DownloadJobRepository,
	boardProfileRepo *db.BoardProfileRepository,
	sourceRepo *db.SourceRepository,
	storage storage.Backend,
) []build.Stage {
	stageList := []build.Stage{
		NewResolveStage(componentRepo, downloadJobRepo, boardProfileRepo, sourceRepo, storage),
		NewDownloadCheckStage(downloadJobRepo, storage),
		NewPrepareStage(storage),
		NewCompileStage(),
		NewAssembleStage(),
		NewPackageStage(storage, 4), // 4GB default image size
	}

	log.Info("Created default build stages",
		"count", len(stageList),
		"stages", []string{"resolve", "download", "prepare", "compile", "assemble", "package"})

	return stageList
}
