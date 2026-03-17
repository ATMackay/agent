package constants

var (
	//
	// https://icinga.com/blog/2022/05/25/embedding-git-commit-information-in-go-binaries/
	//
	Version    = "0.0.0" // overwritten by -ldflag "-X 'github.com/ATMackay/agent/constants.Version=$version'"
	CommitDate = ""      // overwritten by -ldflag "-X 'github.com/ATMackay/agent/constants.CommitDate=$commit_date'"
	GitCommit  = ""      // overwritten by -ldflag "-X 'github.com/ATMackay/agent/constants.GitCommit=$commit_hash'"
	BuildDate  = ""      // overwritten by -ldflag "-X 'github.com/ATMackay/agent/constants.BuildDate=$build_date'"
	Dirty      = "false" // overwritten by -ldflag "-X 'github.com/ATMackay/agent/constants.Dirty=$dirty'"
)
