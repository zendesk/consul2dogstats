package version

var (
	// GitRevision is the Git revision of this build (ref)
	GitRevision string

	// GitDescribe is the Git description of this build (tag or tag + ref)
	GitDescribe string
)
