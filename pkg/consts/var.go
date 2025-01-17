package consts

var PUBLIC_CLASSROOM = ""
var TEACHER_CLASSROOM = ""
var AiTrainSystemNamespace = ""
var NS_prefix = ""

func Init(prefix string) {
	PUBLIC_CLASSROOM = prefix + "-public"
	TEACHER_CLASSROOM = prefix + "-teacher"
	AiTrainSystemNamespace = prefix + "-system"
	NS_prefix = prefix
}
