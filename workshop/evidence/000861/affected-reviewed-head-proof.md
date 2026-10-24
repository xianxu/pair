# Exact reviewed-head planner qualification

Pair HEAD: c7723fbda1471ee2ddb1f13bdc8c30e627dbf88d
Provider: github.com/xianxu/ariadne v0.0.0-20261024032245-5a17f789619d
Provider directory: /Users/xianxu/go/pkg/mod/github.com/xianxu/ariadne@v0.0.0-20261024032245-5a17f789619d
Runtime: go version go1.27.3 darwin/arm64

Command:

    GOPROXY=off GOSUMDB=off ARIADNE_AFFECTED_SOURCE=/Users/xianxu/go/pkg/mod/github.com/xianxu/ariadne@v0.0.0-20261024032245-5a17f789619d python3 -B tests/affected-config-test.py

Exit: 0. Python total: 22.766s. Go fixture: 21.939s.
TestDeclaredPreparationQualifiesCleanPairGraph: PASS 14.73s.
TestActualFileConsumerCoverage: checked 778 distinct group/file inputs.
Raw log: affected-reviewed-head-passed.log
SHA256: 1f50a9d270c96bd5ebc4f0be0e07a9e2e63218ba8bb795ab5c40713d908b3ba4

The existing budgets and runtime source are unchanged. This completes the requested host qualification; it does not explain or negate the review sandbox's timeout.
