package launcher

import "github.com/xianxu/pair/cmd/internal/zellijquery"

var errZellijCrossedReply = zellijquery.ErrCrossedReply
var errZellijSessionLookupMiss = zellijquery.ErrSessionLookupMiss

type zellijReplyKind = zellijquery.ReplyKind

const (
	zellijUnknownReply  = zellijquery.UnknownReply
	zellijSessionsReply = zellijquery.SessionsReply
	zellijClientsReply  = zellijquery.ClientsReply
	zellijPanesReply    = zellijquery.PanesReply
	zellijTabReply      = zellijquery.TabReply
)

func zellijReadKind(args []string) zellijReplyKind { return zellijquery.ReadKind(args) }
func crossedZellijReply(want zellijReplyKind, raw []byte) bool {
	return zellijquery.CrossedReply(want, raw)
}
func retryableSessionLookupMiss(args []string, stdout, stderr []byte, exit int, socketPresent bool) bool {
	return zellijquery.RetryableSessionLookupMiss(args, stdout, stderr, exit, socketPresent)
}
func sessionSocketPresent(name string) bool { return zellijquery.SessionSocketPresent(name) }
