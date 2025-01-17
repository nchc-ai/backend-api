package consts

const NamespaceLabelInstance = "instance"
const DatasetPVCPrefix = "dataset-"

const BaseDockerHubUrl = "https://registry.hub.docker.com/v2/repositories/"
const AiTrainUser = "nchcai"
const AiTrainImagePrefix = "train"

//const PUBLIC_CLASSROOM = "aitrain-public"
//const TEACHER_CLASSROOM = "aitrain-teacher"
//const AiTrainSystemNamespace = "aitrain-system"

// todo: configurable parameter
const TlsSecretName = "nchc-tls-secret"

const SccRoleName = "scc-role"
const SccRoleBindingName = "scc-role-binding"

const LOGIN_ERROR = "登入失敗: "
const (
	ERROR_LOGIN_ROLE_NOT_FOUND = LOGIN_ERROR + "帳號 {%s} 查無對應的身份，請先註冊您的帳號為學生/老師/管理員之一"
)

// Job launch error message format
const JOB_LAUNCH_ERROR = "啟動課程失敗: "

const (
	ERROR_JOB_LAUNCH_QUOTA_FMT    = JOB_LAUNCH_ERROR + "同時間只能啟用1個課程，但您 {%s} 已經啟動 {%d} 個課程"
	ERROR_JOB_LAUNCH_OWNER_FMT    = JOB_LAUNCH_ERROR + "開課列表內課程只能由建立者啟動，但您 {%s} 不是課程建立者"
	ERROR_JOB_LAUNCH_TIME_FMT     = JOB_LAUNCH_ERROR + "教室 {%s} 的課程只能在 {%s} 啟動，現在不是允許的使用時間"
	ERROR_JOB_LAUNCH_MEMBER_FMT   = JOB_LAUNCH_ERROR + "只有成員可以啟動教室內課程，但您 {%s} 並不屬於教室 {%s}"
	ERROR_JOB_LAUNCH_PORT_FMT     = JOB_LAUNCH_ERROR + "課程 {%s} 沒有定義所需要端口，請洽 {%s} 修改設定"
	ERROR_JOB_LAUNCH_BUILDCRD_FMT = JOB_LAUNCH_ERROR + "讀取課程 {%s} 參數錯誤"
	ERROR_JOB_LAUNCH_RUNCRD_FMT   = JOB_LAUNCH_ERROR + "啟動課程 {%s} 後台資源系統出錯"
)

const CLASSROOM_CREATE_ERROR = "教室建立失敗: "
const CLASSROOM_UPDATE_ERROR = "更新教室失敗: "
const CLASSROOM_DELETE_ERROR = "刪除教室失敗: "

// classroom create error message format
const (
	ERROR_CLASSROOM_CREATE_INFO_FMT     = CLASSROOM_CREATE_ERROR + "新增教室 {%s} 基本資訊失敗"
	ERROR_CLASSROOM_CREATE_COURSE_FMT   = CLASSROOM_CREATE_ERROR + "新增教室 {%s} 課程資訊失敗"
	ERROR_CLASSROOM_CREATE_SCHEDULE_FMT = CLASSROOM_CREATE_ERROR + "新增教室 {%s} 允許時用時間資訊失敗"
	ERROR_CLASSROOM_CREATE_TEACHER_FMT  = CLASSROOM_CREATE_ERROR + "新增教室 {%s} 老師資訊失敗"
	ERROR_CLASSROOM_CREATE_STUDENT_FMT  = CLASSROOM_CREATE_ERROR + "新增教室 {%s} 學生資訊失敗"
	ERROR_CLASSROOM_CREATE_CALENDAR_FMT = CLASSROOM_CREATE_ERROR + "新增教室 {%s} 日曆資訊失敗"
	ERROR_CLASSROOM_CREATE_NS_FMT       = CLASSROOM_CREATE_ERROR + "建立教室 {%s} 命名空間失敗"
	ERROR_CLASSROOM_CREATE_DATASET_FMT  = CLASSROOM_CREATE_ERROR + "建立教室 {%s} 資料集失敗"
	ERROR_CLASSROOM_CREATE_SECRET_FMT   = CLASSROOM_CREATE_ERROR + "建立教室 {%s} 憑證失敗"
	ERROR_CLASSROOM_CREATE_ROLE_FMT     = CLASSROOM_CREATE_ERROR + "建立教室 {%s} 權限失敗"
)

// classroom update error message format
const (
	ERROR_CLASSROOM_UPDATE_INFO_FMT     = CLASSROOM_UPDATE_ERROR + "更新教室 {%s} 基本資訊失敗"
	ERROR_CLASSROOM_UPDATE_STUDENT_FMT  = CLASSROOM_UPDATE_ERROR + "更新教室 {%s} 學生資訊失敗"
	ERROR_CLASSROOM_UPDATE_TEACHER_FMT  = CLASSROOM_UPDATE_ERROR + "更新教室 {%s} 老師資訊失敗"
	ERROR_CLASSROOM_UPDATE_SCHEDULE_FMT = CLASSROOM_UPDATE_ERROR + "更新教室 {%s} 允許使用時間資訊失敗"
	ERROR_CLASSROOM_UPDATE_COURSE_FMT   = CLASSROOM_UPDATE_ERROR + "更新教室 {%s} 課程資訊失敗"
	ERROR_CLASSROOM_UPDATE_CALENDAR_FMT = CLASSROOM_UPDATE_ERROR + "更新教室 {%s} 日曆資訊失敗"
)

// classroom delete error message format
const (
	ERROR_CLASSROOM_DELETE_NS_FMT      = CLASSROOM_DELETE_ERROR + "刪除教室 {%s} 後台命名空間失敗"
	ERROR_CLASSROOM_DELETE_DATASET_FMT = CLASSROOM_DELETE_ERROR + "刪除教室 {%s} 後台資料集失敗"
	ERROR_CLASSROOM_DELETE_DEFAULT_FMT = CLASSROOM_DELETE_ERROR + "系統不允許刪除教室 {%s}"
)

const COURSE_CREATE_ERROR = "課程建立失敗: "
const COURSE_UPDATE_ERROR = "更新課程失敗: "
const COURSE_DELETE_ERROR = "刪除課程失敗: "

// course create error message format
const (
	ERROR_COURSE_CREATE_INFO_FMT        = COURSE_CREATE_ERROR + "課程 {%s} 建立課程基本資訊失敗"
	ERROR_COURSE_CREATE_DATASET_FMT     = COURSE_CREATE_ERROR + "課程 {%s} 建立課程資料集資訊失敗"
	ERROR_COURSE_CREATE_PORT_EMPTY_FMT  = COURSE_CREATE_ERROR + "課程 {%s} 建立課程端口資訊失敗，端口名稱為空"
	ERROR_COURSE_CREATE_PORT_INVLID_FMT = COURSE_CREATE_ERROR + "課程 {%s} 建立課程端口資訊失敗，端口名稱不合法"
	ERROR_COURSE_CREATE_PORT_FMT        = COURSE_CREATE_ERROR + "課程 {%s} 建立課程端口資訊失敗"
)

// course update error message format
const (
	ERROR_COURSE_UPDATE_INFO_FMT        = COURSE_UPDATE_ERROR + "課程 {%s} 基本資訊失敗"
	ERROR_COURSE_UPDATE_DATASET_FMT     = COURSE_UPDATE_ERROR + "課程 {%s} 資料集資訊失敗"
	ERROR_COURSE_UPDATE_PORT_FMT        = COURSE_UPDATE_ERROR + "課程 {%s} 端口資訊失敗"
	ERROR_COURSE_UPDATE_PORT_EMPTY_FMT  = COURSE_UPDATE_ERROR + "課程 {%s} 端口資訊失敗，端口名稱為空"
	ERROR_COURSE_UPDATE_PORT_INVLID_FMT = COURSE_UPDATE_ERROR + "課程 {%s} 端口資訊失敗，端口名稱不合法"
)

// course delete error message format
const (
	ERROR_COURSE_DELETE_INFO_FMT = COURSE_DELETE_ERROR + "刪除課程 {%s} 資本資訊失敗"
	ERROR_COURSE_DELETE_JOB_FMT  = COURSE_DELETE_ERROR + "刪除運行中的課程 {%s} 失敗"
)
