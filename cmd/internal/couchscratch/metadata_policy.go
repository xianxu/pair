package couchscratch

// MetadataRoute tags the observed route without granting authority from links.
type MetadataRoute uint8

const (
	MetadataMissing MetadataRoute = iota
	MetadataDirectoryRoute
	MetadataOwnedLink
	MetadataForeignRoute
)

// MetadataState is one startup observation; Unknown and Live override routing.
type MetadataState struct {
	Route                                MetadataRoute
	UseRAM, Backup, Empty, Live, Unknown bool
}

// MetadataAction describes the bounded effect authorized by a metadata state.
type MetadataAction uint8

const (
	MetadataKeep MetadataAction = iota
	MetadataDefer
	MetadataRefuse
	MetadataBackupAndLink
	MetadataCreateLink
	MetadataReplaceEmptyWithLink
	MetadataCreateDirectory
)

// DecideMetadata keeps uncertainty and conflicting paths unchanged.
func DecideMetadata(s MetadataState) MetadataAction {
	if s.Unknown {
		return MetadataRefuse
	}
	if s.Live {
		return MetadataDefer
	}
	if s.Route == MetadataForeignRoute {
		return MetadataRefuse
	}
	if !s.UseRAM {
		if s.Route == MetadataMissing || s.Route == MetadataOwnedLink {
			return MetadataCreateDirectory
		}
		return MetadataKeep
	}
	switch s.Route {
	case MetadataMissing:
		return MetadataCreateLink
	case MetadataOwnedLink:
		return MetadataKeep
	case MetadataDirectoryRoute:
		if !s.Backup {
			return MetadataBackupAndLink
		}
		if s.Empty {
			return MetadataReplaceEmptyWithLink
		}
	}
	return MetadataRefuse
}
