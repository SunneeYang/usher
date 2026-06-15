package config

// DefaultVideoExtensions 内置默认视频扩展名，覆盖常见容器、广播流、摄像机与录像格式。
// 配置文件省略 video_extensions 时将使用此列表。
var DefaultVideoExtensions = []string{
	// 常见容器
	".mp4", ".m4v", ".mkv", ".avi", ".mov", ".wmv", ".flv", ".f4v", ".webm",
	// MPEG 系列
	".mpg", ".mpeg", ".mpe", ".mp2", ".m2v", ".mpv",
	// 广播 / 录制流
	".ts", ".m2ts", ".mts",
	// DVD / 蓝光相关单文件
	".vob", ".iso",
	// Ogg / 开源容器
	".ogv", ".ogm",
	// 移动端 / 网络流
	".3gp", ".3g2", ".asf",
	// RealMedia / 老格式
	".rm", ".rmvb",
	// 编码器命名
	".divx", ".xvid",
	// VCD / 摄像机
	".dat", ".mod", ".tod", ".dv",
	// 专业 / 广播
	".mxf",
	// 电视录制
	".wtv", ".dvr-ms", ".rec", ".str",
	// 裸流（部分 NAS 转码/下载产物）
	".264", ".265", ".h264", ".h265", ".hevc",
}
