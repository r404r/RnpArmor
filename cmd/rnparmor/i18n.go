package main

import (
	"fmt"
	"os"
	"strings"
)

type Lang string

const (
	LangEN Lang = "en"
	LangZH Lang = "zh"
	LangJA Lang = "ja"
)

var langOrder = []Lang{LangEN, LangZH, LangJA}

var langLabels = []string{"English", "中文", "日本語"}

func LangByIndex(i int) Lang {
	if i < 0 || i >= len(langOrder) {
		return LangEN
	}
	return langOrder[i]
}

func LangIndex(l Lang) int {
	for i := range langOrder {
		if langOrder[i] == l {
			return i
		}
	}
	return 0
}

func LangChoices() []string { return langLabels }

func ResolveLang(flagValue string) Lang {
	if l, ok := parseLang(flagValue); ok {
		return l
	}
	if l, ok := parseLang(os.Getenv("RNPARMOR_LANG")); ok {
		return l
	}
	if l, ok := parseLang(os.Getenv("LANG")); ok {
		return l
	}
	return LangEN
}

func parseLang(v string) (Lang, bool) {
	v = strings.TrimSpace(strings.ToLower(v))
	if v == "" {
		return "", false
	}
	v = strings.ReplaceAll(v, "_", "-")
	if i := strings.Index(v, "."); i >= 0 {
		v = v[:i]
	}
	if i := strings.Index(v, "@"); i >= 0 {
		v = v[:i]
	}
	switch {
	case v == "en", v == "english", strings.HasPrefix(v, "en-"):
		return LangEN, true
	case v == "zh", v == "cn", v == "zh-cn", v == "zh-hans", v == "中文", v == "chinese", strings.HasPrefix(v, "zh-"):
		return LangZH, true
	case v == "ja", v == "jp", v == "ja-jp", v == "日本語", v == "japanese", strings.HasPrefix(v, "ja-"):
		return LangJA, true
	default:
		return "", false
	}
}

func Tr(l Lang, key string) string {
	m, ok := i18nCatalog[key]
	if !ok {
		return key
	}
	if v := strings.TrimSpace(m[l]); v != "" {
		return v
	}
	if v := strings.TrimSpace(m[LangEN]); v != "" {
		return v
	}
	return key
}

func ValidateCatalog() error {
	for k, m := range i18nCatalog {
		for _, l := range langOrder {
			if strings.TrimSpace(m[l]) == "" {
				return fmt.Errorf("i18n missing translation: key=%s lang=%s", k, l)
			}
		}
	}
	return nil
}

var i18nCatalog = map[string]map[Lang]string{
	"title.app": {
		LangEN: "RnpArmor",
		LangZH: "RnpArmor",
		LangJA: "RnpArmor",
	},
	"state.idle": {
		LangEN: "idle",
		LangZH: "空闲",
		LangJA: "待機",
	},
	"state.busy": {
		LangEN: "busy",
		LangZH: "忙碌",
		LangJA: "実行中",
	},
	"state.done": {
		LangEN: "done",
		LangZH: "完成",
		LangJA: "完了",
	},
	"status.ready": {
		LangEN: "Ready",
		LangZH: "就绪",
		LangJA: "準備完了",
	},
	"menu.file": {
		LangEN: "File",
		LangZH: "文件",
		LangJA: "ファイル",
	},
	"menu.file.close": {
		LangEN: "Close Window",
		LangZH: "关闭窗口",
		LangJA: "ウィンドウを閉じる",
	},
	"menu.file.quit": {
		LangEN: "Quit",
		LangZH: "退出",
		LangJA: "終了",
	},
	"menu.workspace": {
		LangEN: "Workspace",
		LangZH: "工作区",
		LangJA: "ワークスペース",
	},
	"menu.workspace.keys": {
		LangEN: "Keys",
		LangZH: "钥匙圈与生成",
		LangJA: "鍵リングと生成",
	},
	"menu.workspace.text": {
		LangEN: "Text Studio",
		LangZH: "文本加解密与签名",
		LangJA: "テキスト暗号/署名",
	},
	"menu.workspace.file": {
		LangEN: "File Workbench",
		LangZH: "文件加解密与签名",
		LangJA: "ファイル暗号/署名",
	},
	"menu.workspace.packet": {
		LangEN: "Packet & Features",
		LangZH: "Packet 检视与算法特性",
		LangJA: "パケット解析と機能",
	},
	"menu.workspace.portable": {
		LangEN: "Portable Keyring",
		LangZH: "便携钥匙圈存储",
		LangJA: "ポータブル鍵リング",
	},
	"menu.language": {
		LangEN: "Language",
		LangZH: "语言",
		LangJA: "言語",
	},
	"menu.help": {
		LangEN: "Help",
		LangZH: "帮助",
		LangJA: "ヘルプ",
	},
	"menu.help.about": {
		LangEN: "About",
		LangZH: "关于",
		LangJA: "概要",
	},
	"label.language": {
		LangEN: "Language",
		LangZH: "语言",
		LangJA: "言語",
	},
	"btn.about": {
		LangEN: "About",
		LangZH: "关于",
		LangJA: "概要",
	},
	"title.about": {
		LangEN: "About RnpArmor",
		LangZH: "关于 RnpArmor",
		LangJA: "RnpArmor について",
	},
	"btn.back_workspaces": {
		LangEN: "← Back",
		LangZH: "← 返回工作区",
		LangJA: "← 戻る",
	},
	"about.no_menu_hint": {
		LangEN: "Standalone About window (no menu): workspace controls are intentionally hidden.",
		LangZH: "独立 About 子窗口（无菜单）：工作区功能控件在此视图中被隐藏。",
		LangJA: "独立した About 子ウィンドウ（メニューなし）: この画面ではワークスペース機能を非表示にしています。",
	},
	"btn.close": {
		LangEN: "Close",
		LangZH: "关闭",
		LangJA: "閉じる",
	},
	"about.app_version": {
		LangEN: "Application Version",
		LangZH: "应用版本",
		LangJA: "アプリケーションバージョン",
	},
	"about.author": {
		LangEN: "Author",
		LangZH: "作者",
		LangJA: "作成者",
	},
	"about.repo_url": {
		LangEN: "Repository URL",
		LangZH: "仓库地址",
		LangJA: "リポジトリ URL",
	},
	"about.rnp_upstream": {
		LangEN: "RNP Upstream & Version",
		LangZH: "RNP 地址与版本",
		LangJA: "RNP アップストリームとバージョン",
	},
	"about.local_lib_path": {
		LangEN: "Local RNP Library Path",
		LangZH: "本地 RNP 动态库路径",
		LangJA: "ローカル RNP ライブラリパス",
	},
	"about.keyring_dir": {
		LangEN: "Keyring Directory (portable workspace)",
		LangZH: "密钥库目录（便携工作区）",
		LangJA: "鍵リングディレクトリ（ポータブルワークスペース）",
	},
	"about.pubring_exists": {
		LangEN: "pubring.gpg exists",
		LangZH: "pubring.gpg 存在",
		LangJA: "pubring.gpg の存在",
	},
	"about.secring_exists": {
		LangEN: "secring.gpg exists",
		LangZH: "secring.gpg 存在",
		LangJA: "secring.gpg の存在",
	},
	"about.runtime": {
		LangEN: "UI Framework & Runtime",
		LangZH: "GUI 框架与运行时",
		LangJA: "UI フレームワークとランタイム",
	},
	"about.explicit_keyring_note": {
		LangEN: "RnpArmor uses explicit portable keyring files and does not read the system default keyring.",
		LangZH: "RnpArmor 仅使用显式便携钥匙圈文件，不读取系统默认钥匙圈。",
		LangJA: "RnpArmor は明示的なポータブル鍵リングのみを使用し、システム既定の鍵リングは読み込みません。",
	},
	"about.yes": {
		LangEN: "Yes",
		LangZH: "是",
		LangJA: "はい",
	},
	"about.no": {
		LangEN: "No",
		LangZH: "否",
		LangJA: "いいえ",
	},
	"tab.keys": {
		LangEN: "① Keys",
		LangZH: "① 钥匙圈与生成",
		LangJA: "① 鍵リングと生成",
	},
	"tab.text": {
		LangEN: "② Text Studio",
		LangZH: "② 文本加解密与签名",
		LangJA: "② テキスト暗号/署名",
	},
	"tab.file": {
		LangEN: "③ File Workbench",
		LangZH: "③ 文件加解密与签名",
		LangJA: "③ ファイル暗号/署名",
	},
	"tab.packet": {
		LangEN: "④ Packet & Features",
		LangZH: "④ Packet 检视与算法特性",
		LangJA: "④ パケット解析と機能",
	},
	"tab.portable": {
		LangEN: "⑤ Portable Keyring",
		LangZH: "⑤ 便携钥匙圈存储",
		LangJA: "⑤ ポータブル鍵リング",
	},
	"label.passphrase": {
		LangEN: "Passphrase",
		LangZH: "口令",
		LangJA: "パスフレーズ",
	},
	"label.workspace": {
		LangEN: "Workspace",
		LangZH: "工作目录",
		LangJA: "ワークスペース",
	},
	"label.user_id": {
		LangEN: "User ID",
		LangZH: "用户 ID",
		LangJA: "ユーザー ID",
	},
	"label.primary": {
		LangEN: "Primary Key",
		LangZH: "主密钥",
		LangJA: "主鍵",
	},
	"label.subkey": {
		LangEN: "Subkey",
		LangZH: "子密钥",
		LangJA: "副鍵",
	},
	"label.crypto": {
		LangEN: "Crypto",
		LangZH: "密码参数",
		LangJA: "暗号パラメータ",
	},
	"label.keyring": {
		LangEN: "Keyring",
		LangZH: "钥匙圈",
		LangJA: "鍵リング",
	},
	"label.selected": {
		LangEN: "Selected",
		LangZH: "已选择",
		LangJA: "選択中",
	},
	"placeholder.session_passphrase": {
		LangEN: "session passphrase",
		LangZH: "会话口令",
		LangJA: "セッション用パスフレーズ",
	},
	"placeholder.workspace": {
		LangEN: "portable keyring directory",
		LangZH: "便携钥匙圈目录",
		LangJA: "ポータブル鍵リングのディレクトリ",
	},
	"placeholder.curve": {
		LangEN: "curve",
		LangZH: "曲线",
		LangJA: "曲線",
	},
	"placeholder.key_output": {
		LangEN: "key JSON / export output",
		LangZH: "密钥 JSON / 导出结果",
		LangJA: "鍵 JSON / エクスポート結果",
	},
	"placeholder.result_output": {
		LangEN: "result output",
		LangZH: "结果输出",
		LangJA: "結果出力",
	},
	"placeholder.feature_query": {
		LangEN: "empty, or type, or type:name",
		LangZH: "留空，或类型，或 类型:名称",
		LangJA: "空欄、または type、または type:name",
	},
	"btn.open_session": {
		LangEN: "Open session",
		LangZH: "打开会话",
		LangJA: "セッション開始",
	},
	"btn.generate_key": {
		LangEN: "Generate key",
		LangZH: "生成密钥",
		LangJA: "鍵生成",
	},
	"btn.reload_keys": {
		LangEN: "Reload keys",
		LangZH: "刷新密钥",
		LangJA: "鍵を再読込",
	},
	"btn.save_keyring": {
		LangEN: "Save keyring",
		LangZH: "保存钥匙圈",
		LangJA: "鍵リング保存",
	},
	"btn.load_keyring": {
		LangEN: "Load keyring",
		LangZH: "加载钥匙圈",
		LangJA: "鍵リング読込",
	},
	"btn.key_json": {
		LangEN: "Key JSON",
		LangZH: "密钥 JSON",
		LangJA: "鍵 JSON",
	},
	"btn.export_public": {
		LangEN: "Export public",
		LangZH: "导出公钥",
		LangJA: "公開鍵をエクスポート",
	},
	"btn.export_secret": {
		LangEN: "Export secret",
		LangZH: "导出私钥",
		LangJA: "秘密鍵をエクスポート",
	},
	"btn.delete_key": {
		LangEN: "Delete key",
		LangZH: "删除密钥",
		LangJA: "鍵削除",
	},
	"label.plain_text": {
		LangEN: "Plain text",
		LangZH: "明文",
		LangJA: "平文",
	},
	"label.recipients": {
		LangEN: "Recipients (comma fp)",
		LangZH: "收件人（逗号分隔指纹）",
		LangJA: "受信者（カンマ区切り指紋）",
	},
	"label.password": {
		LangEN: "Password",
		LangZH: "密码",
		LangJA: "パスワード",
	},
	"label.cipher_compression": {
		LangEN: "Cipher / Compression",
		LangZH: "加密算法 / 压缩",
		LangJA: "暗号方式 / 圧縮",
	},
	"label.sign_mode": {
		LangEN: "Sign mode",
		LangZH: "签名模式",
		LangJA: "署名モード",
	},
	"label.signer_fingerprint": {
		LangEN: "Signer fingerprint",
		LangZH: "签名者指纹",
		LangJA: "署名者フィンガープリント",
	},
	"label.detached_verify_sig": {
		LangEN: "Detached verify sig",
		LangZH: "分离验签签名",
		LangJA: "Detached 検証シグネチャ",
	},
	"toggle.encrypt_sign": {
		LangEN: "Encrypt+Sign",
		LangZH: "加密并签名",
		LangJA: "暗号化+署名",
	},
	"btn.encrypt_text": {
		LangEN: "Encrypt text",
		LangZH: "加密文本",
		LangJA: "テキスト暗号化",
	},
	"btn.decrypt_text": {
		LangEN: "Decrypt text",
		LangZH: "解密文本",
		LangJA: "テキスト復号",
	},
	"btn.sign_text": {
		LangEN: "Sign text",
		LangZH: "文本签名",
		LangJA: "テキスト署名",
	},
	"btn.verify_text": {
		LangEN: "Verify text",
		LangZH: "验签文本",
		LangJA: "テキスト検証",
	},
	"hint.text_alg": {
		LangEN: "Algorithm note: cipher/compression apply to encryption; sign mode chooses clearsign, detached, or inline.",
		LangZH: "算法说明：加密时使用密码算法/压缩；签名模式可切换明文签名、分离签名或内联签名。",
		LangJA: "アルゴリズム説明: 暗号化では暗号方式/圧縮が適用され、署名モードで clearsign / detached / inline を選択します。",
	},
	"hint.verify_badge": {
		LangEN: "Verification badges: mathematical validity is cryptographic only; identity trust is not asserted.",
		LangZH: "验签双徽章：数学有效仅表示密码学正确，不代表身份可信。",
		LangJA: "検証バッジ: 数学的有効性は暗号学的整合のみを示し、本人性の信頼は保証しません。",
	},
	"hint.keys_alg": {
		LangEN: "Algorithm note: key generation supports multiple public-key algorithms via librnp JSON generation and explicit session keyring management.",
		LangZH: "算法说明：密钥生成通过 librnp JSON 入口支持多种公钥算法，并采用显式会话钥匙圈管理。",
		LangJA: "アルゴリズム説明: 鍵生成は librnp の JSON 生成経路で複数の公開鍵アルゴリズムを扱い、明示的なセッション鍵リング管理を行います。",
	},
	"label.source": {
		LangEN: "Source",
		LangZH: "源文件",
		LangJA: "入力ファイル",
	},
	"label.destination": {
		LangEN: "Destination",
		LangZH: "目标文件",
		LangJA: "出力ファイル",
	},
	"label.signature_path": {
		LangEN: "Signature path",
		LangZH: "签名文件路径",
		LangJA: "署名ファイルパス",
	},
	"label.detached_verify": {
		LangEN: "Detached verify",
		LangZH: "分离验签",
		LangJA: "Detached 検証",
	},
	"toggle.detached": {
		LangEN: "Detached",
		LangZH: "分离",
		LangJA: "Detached",
	},
	"btn.encrypt_file": {
		LangEN: "Encrypt file",
		LangZH: "加密文件",
		LangJA: "ファイル暗号化",
	},
	"btn.decrypt_file": {
		LangEN: "Decrypt file",
		LangZH: "解密文件",
		LangJA: "ファイル復号",
	},
	"btn.sign_file": {
		LangEN: "Sign file",
		LangZH: "文件签名",
		LangJA: "ファイル署名",
	},
	"btn.verify_file": {
		LangEN: "Verify file",
		LangZH: "验签文件",
		LangJA: "ファイル検証",
	},
	"hint.file_alg": {
		LangEN: "File mode keeps non-overwrite semantics; choose detached/inline signing mode from Sign mode.",
		LangZH: "文件模式保持防覆盖语义；签名模式支持分离/内联。",
		LangJA: "ファイルモードは既存ファイル非上書きを維持し、署名モードで detached / inline を選択できます。",
	},
	"label.packet_input_file": {
		LangEN: "Packet input file",
		LangZH: "Packet 输入文件",
		LangJA: "パケット入力ファイル",
	},
	"label.as_json": {
		LangEN: "As JSON",
		LangZH: "JSON 输出",
		LangJA: "JSON 出力",
	},
	"toggle.json": {
		LangEN: "JSON",
		LangZH: "JSON",
		LangJA: "JSON",
	},
	"label.feature_query": {
		LangEN: "Feature query",
		LangZH: "特性查询",
		LangJA: "機能クエリ",
	},
	"btn.dump_packets": {
		LangEN: "Dump packets",
		LangZH: "解析 Packet",
		LangJA: "パケット解析",
	},
	"btn.query_features": {
		LangEN: "Query features",
		LangZH: "查询算法特性",
		LangJA: "機能問い合わせ",
	},
	"label.packet_output": {
		LangEN: "Packet output",
		LangZH: "Packet 输出",
		LangJA: "パケット出力",
	},
	"label.features": {
		LangEN: "Features",
		LangZH: "特性",
		LangJA: "機能",
	},
	"hint.packet_alg": {
		LangEN: "Packet inspector and feature query can audit supported algorithms and message structure.",
		LangZH: "Packet 检视与特性查询可用于审计消息结构与算法支持矩阵。",
		LangJA: "パケット解析と機能問い合わせにより、メッセージ構造と対応アルゴリズムを監査できます。",
	},
	"btn.open_load": {
		LangEN: "Open+Load",
		LangZH: "打开并加载",
		LangJA: "開始+読込",
	},
	"btn.save": {
		LangEN: "Save",
		LangZH: "保存",
		LangJA: "保存",
	},
	"note.portable_files": {
		LangEN: "This workspace stores only explicit keyring files (pubring.gpg / secring.gpg).",
		LangZH: "该工作区仅存储显式钥匙圈文件（pubring.gpg / secring.gpg）。",
		LangJA: "このワークスペースは明示的な鍵リングファイル（pubring.gpg / secring.gpg）のみ保存します。",
	},
	"note.portable_pass": {
		LangEN: "Passphrase is used only in memory for this session.",
		LangZH: "口令仅在当前会话内存中使用。",
		LangJA: "パスフレーズはこのセッションのメモリ内でのみ使用されます。",
	},
	"err.workspace_empty": {
		LangEN: "workspace is empty",
		LangZH: "工作目录为空",
		LangJA: "ワークスペースが空です",
	},
	"err.no_key_selected": {
		LangEN: "no key selected",
		LangZH: "未选择密钥",
		LangJA: "鍵が選択されていません",
	},
	"verify.math_valid": {
		LangEN: "[Math: VALID]",
		LangZH: "[数学有效: 是]",
		LangJA: "[数学的有効: はい]",
	},
	"verify.math_invalid": {
		LangEN: "[Math: INVALID]",
		LangZH: "[数学有效: 否]",
		LangJA: "[数学的有効: いいえ]",
	},
	"verify.identity_unverified": {
		LangEN: "[Identity: UNVERIFIED]",
		LangZH: "[身份: 未验证]",
		LangJA: "[本人性: 未検証]",
	},
	"verify.identity_trusted": {
		LangEN: "[Identity: TRUSTED]",
		LangZH: "[身份: 已信任]",
		LangJA: "[本人性: 信頼済み]",
	},
	"verify.status": {
		LangEN: "Status",
		LangZH: "状态",
		LangJA: "ステータス",
	},
	"verify.keyid": {
		LangEN: "KeyID",
		LangZH: "KeyID",
		LangJA: "KeyID",
	},
	"verify.fingerprint": {
		LangEN: "Fingerprint",
		LangZH: "指纹",
		LangJA: "フィンガープリント",
	},
	"verify.hash": {
		LangEN: "Hash",
		LangZH: "哈希",
		LangJA: "ハッシュ",
	},
	"verify.created": {
		LangEN: "Created",
		LangZH: "创建时间",
		LangJA: "作成時刻",
	},
	"verify.expires": {
		LangEN: "Expires",
		LangZH: "过期时间",
		LangJA: "有効期限",
	},
	"action.open_session": {
		LangEN: "Open session",
		LangZH: "打开会话",
		LangJA: "セッション開始",
	},
	"action.generate_key": {
		LangEN: "Generate key",
		LangZH: "生成密钥",
		LangJA: "鍵生成",
	},
	"action.list_keys": {
		LangEN: "Reload keys",
		LangZH: "刷新密钥",
		LangJA: "鍵を再読込",
	},
	"action.save_keyring": {
		LangEN: "Save keyring",
		LangZH: "保存钥匙圈",
		LangJA: "鍵リング保存",
	},
	"action.load_keyring": {
		LangEN: "Load keyring",
		LangZH: "加载钥匙圈",
		LangJA: "鍵リング読込",
	},
	"action.key_json": {
		LangEN: "Key JSON",
		LangZH: "密钥 JSON",
		LangJA: "鍵 JSON",
	},
	"action.export_public": {
		LangEN: "Export public",
		LangZH: "导出公钥",
		LangJA: "公開鍵をエクスポート",
	},
	"action.export_secret": {
		LangEN: "Export secret",
		LangZH: "导出私钥",
		LangJA: "秘密鍵をエクスポート",
	},
	"action.delete_key": {
		LangEN: "Delete key",
		LangZH: "删除密钥",
		LangJA: "鍵削除",
	},
	"action.encrypt_text": {
		LangEN: "Encrypt text",
		LangZH: "加密文本",
		LangJA: "テキスト暗号化",
	},
	"action.decrypt_text": {
		LangEN: "Decrypt text",
		LangZH: "解密文本",
		LangJA: "テキスト復号",
	},
	"action.sign_text": {
		LangEN: "Sign text",
		LangZH: "文本签名",
		LangJA: "テキスト署名",
	},
	"action.verify_text": {
		LangEN: "Verify text",
		LangZH: "验签文本",
		LangJA: "テキスト検証",
	},
	"action.encrypt_file": {
		LangEN: "Encrypt file",
		LangZH: "加密文件",
		LangJA: "ファイル暗号化",
	},
	"action.decrypt_file": {
		LangEN: "Decrypt file",
		LangZH: "解密文件",
		LangJA: "ファイル復号",
	},
	"action.sign_file": {
		LangEN: "Sign file",
		LangZH: "文件签名",
		LangJA: "ファイル署名",
	},
	"action.verify_file": {
		LangEN: "Verify file",
		LangZH: "验签文件",
		LangJA: "ファイル検証",
	},
	"action.dump_packets": {
		LangEN: "Dump packets",
		LangZH: "解析 Packet",
		LangJA: "パケット解析",
	},
	"action.query_features": {
		LangEN: "Query features",
		LangZH: "查询算法特性",
		LangJA: "機能問い合わせ",
	},
	"action.open_load": {
		LangEN: "Open+Load",
		LangZH: "打开并加载",
		LangJA: "開始+読込",
	},
	"action.save": {
		LangEN: "Save",
		LangZH: "保存",
		LangJA: "保存",
	},
	"action.import_keys": {
		LangEN: "Import keys",
		LangZH: "导入密钥",
		LangJA: "鍵をインポート",
	},
	"action.export_public_file": {
		LangEN: "Export public key",
		LangZH: "导出公钥",
		LangJA: "公開鍵を書き出す",
	},
	"action.export_secret_file": {
		LangEN: "Export secret key",
		LangZH: "导出私钥",
		LangJA: "秘密鍵を書き出す",
	},
	"action.export_revocation": {
		LangEN: "Export revocation certificate",
		LangZH: "导出吊销证书",
		LangJA: "失効証明書を書き出す",
	},
	"action.copy_public": {
		LangEN: "Copy public key",
		LangZH: "复制公钥",
		LangJA: "公開鍵をコピー",
	},
	"action.copy_secret": {
		LangEN: "Copy secret key",
		LangZH: "复制私钥",
		LangJA: "秘密鍵をコピー",
	},
	"btn.choose_folder": {
		LangEN: "Choose Folder...",
		LangZH: "选择文件夹...",
		LangJA: "フォルダを選択...",
	},
	"btn.new_key": {
		LangEN: "+ New Key",
		LangZH: "+ 新建密钥",
		LangJA: "+ 新しい鍵",
	},
	"btn.import_file": {
		LangEN: "Import File...",
		LangZH: "导入文件...",
		LangJA: "ファイルをインポート...",
	},
	"btn.import_text": {
		LangEN: "Import Text...",
		LangZH: "粘贴导入...",
		LangJA: "テキストをインポート...",
	},
	"btn.refresh": {
		LangEN: "Refresh",
		LangZH: "刷新",
		LangJA: "更新",
	},
	"btn.copy_fingerprint": {
		LangEN: "Copy Fingerprint",
		LangZH: "复制指纹",
		LangJA: "指紋をコピー",
	},
	"btn.copy_public": {
		LangEN: "Copy Public Key",
		LangZH: "复制公钥",
		LangJA: "公開鍵をコピー",
	},
	"btn.copy_secret": {
		LangEN: "Copy Secret",
		LangZH: "复制私钥",
		LangJA: "秘密鍵をコピー",
	},
	"btn.export_public_asc": {
		LangEN: "Export Public (.asc)...",
		LangZH: "导出公钥 (.asc)...",
		LangJA: "公開鍵を書き出す (.asc)...",
	},
	"btn.export_secret_asc": {
		LangEN: "Export Secret (.asc)...",
		LangZH: "导出私钥 (.asc)...",
		LangJA: "秘密鍵を書き出す (.asc)...",
	},
	"btn.revocation": {
		LangEN: "Revocation Cert...",
		LangZH: "吊销证书...",
		LangJA: "失効証明書...",
	},
	"btn.delete_key_ellipsis": {
		LangEN: "Delete Key...",
		LangZH: "删除密钥...",
		LangJA: "鍵を削除...",
	},
	"btn.generate": {
		LangEN: "Generate Key",
		LangZH: "生成密钥",
		LangJA: "鍵を生成",
	},
	"btn.import": {
		LangEN: "Import",
		LangZH: "导入",
		LangJA: "インポート",
	},
	"btn.paste_clipboard": {
		LangEN: "Paste from Clipboard",
		LangZH: "从剪贴板粘贴",
		LangJA: "クリップボードから貼り付け",
	},
	"btn.load_json": {
		LangEN: "Load JSON",
		LangZH: "加载 JSON",
		LangJA: "JSON を読み込む",
	},
	"btn.cancel": {
		LangEN: "Cancel",
		LangZH: "取消",
		LangJA: "キャンセル",
	},
	"btn.delete": {
		LangEN: "Delete",
		LangZH: "删除",
		LangJA: "削除",
	},
	"seg.all": {
		LangEN: "All",
		LangZH: "全部",
		LangJA: "すべて",
	},
	"seg.mine": {
		LangEN: "My Keys",
		LangZH: "我的密钥",
		LangJA: "自分の鍵",
	},
	"seg.public": {
		LangEN: "Public",
		LangZH: "公钥",
		LangJA: "公開鍵",
	},
	"seg.id16": {
		LangEN: "Key ID (16)",
		LangZH: "密钥 ID (16)",
		LangJA: "鍵 ID (16)",
	},
	"seg.fp40": {
		LangEN: "Fingerprint (40)",
		LangZH: "指纹 (40)",
		LangJA: "フィンガープリント (40)",
	},
	"col.type": {
		LangEN: "Type",
		LangZH: "类型",
		LangJA: "種類",
	},
	"col.name": {
		LangEN: "Name / Email",
		LangZH: "名称 / 邮箱",
		LangJA: "名前 / メール",
	},
	"col.key_id": {
		LangEN: "Key ID",
		LangZH: "密钥 ID",
		LangJA: "鍵 ID",
	},
	"col.fingerprint": {
		LangEN: "Fingerprint",
		LangZH: "指纹",
		LangJA: "フィンガープリント",
	},
	"col.algorithm": {
		LangEN: "Algorithm",
		LangZH: "算法",
		LangJA: "アルゴリズム",
	},
	"col.status": {
		LangEN: "Status",
		LangZH: "状态",
		LangJA: "状態",
	},
	"badge.secret": {
		LangEN: "Secret",
		LangZH: "私钥",
		LangJA: "秘密鍵",
	},
	"badge.public": {
		LangEN: "Public",
		LangZH: "公钥",
		LangJA: "公開鍵",
	},
	"badge.secret_public": {
		LangEN: "Secret+Public",
		LangZH: "私钥+公钥",
		LangJA: "秘密鍵+公開鍵",
	},
	"badge.valid": {
		LangEN: "Valid",
		LangZH: "有效",
		LangJA: "有効",
	},
	"badge.expired": {
		LangEN: "Expired",
		LangZH: "已过期",
		LangJA: "期限切れ",
	},
	"badge.revoked": {
		LangEN: "Revoked",
		LangZH: "已吊销",
		LangJA: "失効",
	},
	"badge.certify": {
		LangEN: "Certify",
		LangZH: "认证",
		LangJA: "証明",
	},
	"badge.sign": {
		LangEN: "Sign",
		LangZH: "签名",
		LangJA: "署名",
	},
	"badge.encrypt": {
		LangEN: "Encrypt",
		LangZH: "加密",
		LangJA: "暗号化",
	},
	"label.expires": {
		LangEN: "Expires",
		LangZH: "过期",
		LangJA: "有効期限",
	},
	"label.created": {
		LangEN: "Created",
		LangZH: "创建",
		LangJA: "作成",
	},
	"label.never": {
		LangEN: "Never",
		LangZH: "永不",
		LangJA: "無期限",
	},
	"label.key_id": {
		LangEN: "Key ID",
		LangZH: "密钥 ID",
		LangJA: "鍵 ID",
	},
	"label.fingerprint": {
		LangEN: "Fingerprint",
		LangZH: "指纹",
		LangJA: "フィンガープリント",
	},
	"label.subkeys": {
		LangEN: "Subkeys",
		LangZH: "子密钥",
		LangJA: "副鍵",
	},
	"label.raw_json": {
		LangEN: "Raw Key JSON",
		LangZH: "原始密钥 JSON",
		LangJA: "鍵 JSON",
	},
	"label.name": {
		LangEN: "Name",
		LangZH: "名称",
		LangJA: "名前",
	},
	"label.email": {
		LangEN: "Email",
		LangZH: "邮箱",
		LangJA: "メール",
	},
	"label.confirm_pass": {
		LangEN: "Confirm Passphrase",
		LangZH: "确认口令",
		LangJA: "パスフレーズの確認",
	},
	"label.expiration": {
		LangEN: "Expiration",
		LangZH: "有效期",
		LangJA: "有効期限",
	},
	"label.advanced": {
		LangEN: "Advanced algorithms",
		LangZH: "高级算法选项",
		LangJA: "高度なアルゴリズム",
	},
	"label.none": {
		LangEN: "None",
		LangZH: "无",
		LangJA: "なし",
	},
	"placeholder.unlock_pass": {
		LangEN: "Passphrase to unlock secret keys",
		LangZH: "用于解锁私钥的口令",
		LangJA: "秘密鍵を解除するパスフレーズ",
	},
	"placeholder.search": {
		LangEN: "Search UID, Key ID, fingerprint, algorithm",
		LangZH: "搜索 UID、密钥 ID、指纹、算法",
		LangJA: "UID・鍵 ID・指紋・アルゴリズムを検索",
	},
	"placeholder.import_text": {
		LangEN: "Paste an armored public or secret key",
		LangZH: "粘贴装甲公钥或私钥",
		LangJA: "装甲公開鍵または秘密鍵を貼り付け",
	},
	"placeholder.name": {
		LangEN: "Name",
		LangZH: "名称",
		LangJA: "名前",
	},
	"placeholder.email": {
		LangEN: "name@example.com",
		LangZH: "name@example.com",
		LangJA: "name@example.com",
	},
	"placeholder.new_pass": {
		LangEN: "Passphrase",
		LangZH: "口令",
		LangJA: "パスフレーズ",
	},
	"placeholder.confirm_pass": {
		LangEN: "Confirm passphrase",
		LangZH: "再次输入口令",
		LangJA: "パスフレーズを再入力",
	},
	"exp.never": {
		LangEN: "Never",
		LangZH: "永不",
		LangJA: "無期限",
	},
	"exp.1y": {
		LangEN: "1 Year",
		LangZH: "1 年",
		LangJA: "1 年",
	},
	"exp.2y": {
		LangEN: "2 Years",
		LangZH: "2 年",
		LangJA: "2 年",
	},
	"exp.5y": {
		LangEN: "5 Years",
		LangZH: "5 年",
		LangJA: "5 年",
	},
	"err.name_or_email": {
		LangEN: "Enter a name or an email.",
		LangZH: "请至少填写名称或邮箱。",
		LangJA: "名前またはメールを入力してください。",
	},
	"err.pass_empty": {
		LangEN: "Passphrase is required.",
		LangZH: "口令不能为空。",
		LangJA: "パスフレーズは必須です。",
	},
	"err.pass_mismatch": {
		LangEN: "Passphrase and confirmation do not match.",
		LangZH: "两次输入的口令不一致。",
		LangJA: "パスフレーズが一致しません。",
	},
	"err.import_empty": {
		LangEN: "Nothing to import.",
		LangZH: "没有可导入的内容。",
		LangJA: "インポートする内容がありません。",
	},
	"empty.keys": {
		LangEN: "No keys in this folder. Choose a folder or create a key.",
		LangZH: "此文件夹没有密钥。请选择文件夹或新建密钥。",
		LangJA: "このフォルダに鍵はありません。フォルダを選ぶか鍵を作成してください。",
	},
	"empty.search": {
		LangEN: "No keys match this search.",
		LangZH: "没有符合搜索的密钥。",
		LangJA: "検索に一致する鍵はありません。",
	},
	"empty.select": {
		LangEN: "Select a key to inspect it.",
		LangZH: "选择一把密钥以查看详情。",
		LangJA: "鍵を選ぶと詳細を表示します。",
	},
	"empty.subkeys": {
		LangEN: "No subkeys.",
		LangZH: "没有子密钥。",
		LangJA: "副鍵はありません。",
	},
	"dlg.choose_folder": {
		LangEN: "Choose keyring folder",
		LangZH: "选择钥匙圈文件夹",
		LangJA: "鍵リングのフォルダを選択",
	},
	"dlg.import_file": {
		LangEN: "Import key file",
		LangZH: "导入密钥文件",
		LangJA: "鍵ファイルをインポート",
	},
	"dlg.export_public": {
		LangEN: "Export public key",
		LangZH: "导出公钥",
		LangJA: "公開鍵を書き出す",
	},
	"dlg.export_secret": {
		LangEN: "Export secret key",
		LangZH: "导出私钥",
		LangJA: "秘密鍵を書き出す",
	},
	"dlg.export_revocation": {
		LangEN: "Save revocation certificate",
		LangZH: "保存吊销证书",
		LangJA: "失効証明書を保存",
	},
	"dlg.delete_title": {
		LangEN: "Delete key",
		LangZH: "删除密钥",
		LangJA: "鍵を削除",
	},
	"dlg.delete_public": {
		LangEN: "This deletes the public key.",
		LangZH: "这将删除公钥。",
		LangJA: "公開鍵を削除します。",
	},
	"dlg.delete_secret": {
		LangEN: "This deletes the public key and the secret key.",
		LangZH: "这将同时删除公钥和私钥。",
		LangJA: "公開鍵と秘密鍵を削除します。",
	},
	"title.new_key": {
		LangEN: "New Key",
		LangZH: "新建密钥",
		LangJA: "新しい鍵",
	},
	"title.import_text": {
		LangEN: "Import Key Text",
		LangZH: "粘贴导入密钥",
		LangJA: "鍵テキストをインポート",
	},
	"hint.secret_export": {
		LangEN: "A secret key export can unlock your mail and files. The file is saved with mode 0600.",
		LangZH: "导出私钥可能解锁你的邮件和文件。文件将以 0600 权限保存。",
		LangJA: "秘密鍵の書き出しはメールとファイルを解錠できます。ファイルは 0600 で保存します。",
	},
	"btn.add_recipient": {
		LangEN: "+ Add",
		LangZH: "+ 添加",
		LangJA: "+ 追加",
	},
	"btn.clear_recipients": {
		LangEN: "Clear",
		LangZH: "清除",
		LangJA: "クリア",
	},
	"btn.browse": {
		LangEN: "Browse...",
		LangZH: "浏览...",
		LangJA: "参照...",
	},
	"btn.save_as": {
		LangEN: "Save As...",
		LangZH: "另存为...",
		LangJA: "別名で保存...",
	},
	"signer.use_selected": {
		LangEN: "Use selected key (default)",
		LangZH: "使用当前选中密钥（默认）",
		LangJA: "選択中の鍵を使う（既定）",
	},
	"empty.recipient_keys": {
		LangEN: "No keys loaded. Open a keyring, or paste a fingerprint.",
		LangZH: "未加载密钥。请打开钥匙圈，或粘贴指纹。",
		LangJA: "鍵が読み込まれていません。鍵リングを開くか、指紋を貼り付けてください。",
	},
	"placeholder.recipients": {
		LangEN: "Comma-separated fingerprints",
		LangZH: "逗号分隔的指纹，可粘贴",
		LangJA: "カンマ区切りの指紋（貼り付け可）",
	},
	"placeholder.signer": {
		LangEN: "Fingerprint override",
		LangZH: "手动覆盖指纹",
		LangJA: "指紋を手入力で上書き",
	},
	"placeholder.symm_pass": {
		LangEN: "Symmetric passphrase",
		LangZH: "对称口令",
		LangJA: "共通鍵のパスフレーズ",
	},
	"dlg.browse_source": {
		LangEN: "Choose source file",
		LangZH: "选择源文件",
		LangJA: "入力ファイルを選択",
	},
	"dlg.save_destination": {
		LangEN: "Save destination as",
		LangZH: "目标文件另存为",
		LangJA: "出力先を別名で保存",
	},
	"dlg.browse_signature": {
		LangEN: "Choose signature file",
		LangZH: "选择签名文件",
		LangJA: "署名ファイルを選択",
	},
	"dlg.save_signature": {
		LangEN: "Save signature as",
		LangZH: "签名文件另存为",
		LangJA: "署名を別名で保存",
	},
	"dlg.browse_packet": {
		LangEN: "Open OpenPGP message",
		LangZH: "打开 OpenPGP 文件",
		LangJA: "OpenPGP ファイルを開く",
	},
	"filter.openpgp": {
		LangEN: "OpenPGP",
		LangZH: "OpenPGP",
		LangJA: "OpenPGP",
	},
	"filter.all_files": {
		LangEN: "All files",
		LangZH: "所有文件",
		LangJA: "すべてのファイル",
	},
	"label.alg_preset": {
		LangEN: "Algorithm preset",
		LangZH: "算法预设",
		LangJA: "アルゴリズムのプリセット",
	},
	"label.enc_subkey": {
		LangEN: "Encryption Subkey",
		LangZH: "加密子密钥",
		LangJA: "暗号化副鍵",
	},
	"preset.modern": {
		LangEN: "Modern (Ed25519 + Curve25519)",
		LangZH: "现代推荐（Ed25519 + Curve25519）",
		LangJA: "現代的（Ed25519 + Curve25519）",
	},
	"preset.rsa4096": {
		LangEN: "RSA 4096 + RSA 4096",
		LangZH: "RSA 4096 + RSA 4096",
		LangJA: "RSA 4096 + RSA 4096",
	},
	"preset.rsa3072": {
		LangEN: "RSA 3072 + RSA 3072",
		LangZH: "RSA 3072 + RSA 3072",
		LangJA: "RSA 3072 + RSA 3072",
	},
	"preset.nist_p256": {
		LangEN: "NIST P-256",
		LangZH: "NIST P-256（合规）",
		LangJA: "NIST P-256（準拠）",
	},
	"preset.custom": {
		LangEN: "Custom",
		LangZH: "自定义",
		LangJA: "カスタム",
	},
	"preset.modern.desc": {
		LangEN: "Recommended: Ed25519 with Curve25519. Short keys, fast, and widely supported.",
		LangZH: "推荐：Ed25519 配 Curve25519。主密钥短、速度快，现代客户端通用。",
		LangJA: "推奨：Ed25519 と Curve25519。主鍵は短く高速で、新しいクライアントで広く使えます。",
	},
	"preset.rsa4096.desc": {
		LangEN: "Highest compatibility: RSA 4096 for both the primary key and the encryption subkey.",
		LangZH: "最高兼容：主密钥与加密子密钥均为 RSA 4096。",
		LangJA: "高い互換性：主鍵と暗号化副鍵のどちらも RSA 4096。",
	},
	"preset.rsa3072.desc": {
		LangEN: "Balanced compatibility: RSA 3072 for both the primary key and the encryption subkey.",
		LangZH: "兼容与体积均衡：主密钥与加密子密钥均为 RSA 3072。",
		LangJA: "互換とサイズの均衡：主鍵と暗号化副鍵のどちらも RSA 3072。",
	},
	"preset.nist_p256.desc": {
		LangEN: "NIST profile: ECDSA P-256 for the primary key and ECDH P-256 for the encryption subkey.",
		LangZH: "NIST 合规：主密钥为 ECDSA P-256，加密子密钥为 ECDH P-256。",
		LangJA: "NIST 準拠：主鍵は ECDSA P-256、暗号化副鍵は ECDH P-256。",
	},
	"preset.custom.desc": {
		LangEN: "Custom: choose the primary key and encryption subkey algorithms yourself.",
		LangZH: "自定义：自行选择主密钥与加密子密钥算法。",
		LangJA: "カスタム：主鍵と暗号化副鍵のアルゴリズムを自分で選びます。",
	},
	"tip.preset.roles": {
		LangEN: "The primary key certifies and signs. The subkey encrypts. Pick a preset, or customize each one.",
		LangZH: "主密钥负责认证和签名，子密钥负责加密。可选预设，也可分别自定义。",
		LangJA: "主鍵は認証と署名、副鍵は暗号化を担当します。プリセットか個別の指定を選べます。",
	},
	"tip.primary.eddsa": {
		LangEN: "Recommended: short keys, fast, and supported by modern clients.",
		LangZH: "推荐：密钥短、速度快、现代客户端通用。",
		LangJA: "推奨：鍵は短く高速で、新しいクライアントで使えます。",
	},
	"tip.primary.rsa": {
		LangEN: "High compatibility: for older PGP clients or enterprise gateways.",
		LangZH: "高兼容：适合旧版 PGP 客户端或企业网关。",
		LangJA: "高い互換性：古い PGP クライアントや企業ゲートウェイ向け。",
	},
	"tip.primary.ecdsa": {
		LangEN: "NIST elliptic curves, for environments that require them.",
		LangZH: "NIST 标准椭圆曲线，适合合规场景。",
		LangJA: "NIST 標準の楕円曲線。準拠が必要な場面向け。",
	},
	"tip.primary.dsa": {
		LangEN: "A legacy algorithm. Not recommended for new keys.",
		LangZH: "历史兼容算法，不建议用于新密钥。",
		LangJA: "過去の互換用アルゴリズム。新しい鍵には非推奨。",
	},
	"tip.primary.role": {
		LangEN: "Primary key: certifies and signs. The curve or size follows the algorithm.",
		LangZH: "主密钥负责认证和签名。曲线或位数随算法变化。",
		LangJA: "主鍵は認証と署名を担当します。曲線またはビット長はアルゴリズムに従います。",
	},
	"tip.subkey.ecdh": {
		LangEN: "Recommended: Curve25519 is the modern encryption partner for Ed25519.",
		LangZH: "推荐：Curve25519 是 Ed25519 的现代加密搭档。",
		LangJA: "推奨：Curve25519 は Ed25519 の現代的な暗号化の相手です。",
	},
	"tip.subkey.rsa": {
		LangEN: "Compatible with older tools.",
		LangZH: "兼容旧版工具。",
		LangJA: "古いツールとの互換性があります。",
	},
	"tip.subkey.elgamal": {
		LangEN: "A legacy algorithm. Large, and not recommended for new keys.",
		LangZH: "历史遗留算法，体积大，不建议用于新密钥。",
		LangJA: "過去のアルゴリズム。サイズが大きく、新しい鍵には非推奨。",
	},
	"tip.subkey.role": {
		LangEN: "Encryption subkey: encrypts to you. Its curve or size follows the algorithm.",
		LangZH: "加密子密钥负责加密。曲线或位数随算法变化。",
		LangJA: "暗号化副鍵は暗号化を担当します。曲線またはビット長はアルゴリズムに従います。",
	},
	"tip.crypto.desc": {
		LangEN: "Recommended: SHA256 or SHA512, with AES256.",
		LangZH: "推荐使用 SHA256 或 SHA512，对称算法用 AES256。",
		LangJA: "推奨は SHA256 または SHA512 と AES256 です。",
	},
	"tip.crypto.role": {
		LangEN: "Hash and cipher protect the secret key and symmetric encryption. SHA256 or SHA512 with AES256 is the usual pair.",
		LangZH: "哈希与对称算法用于保护私钥和对称加密。通常搭配 SHA256 或 SHA512 与 AES256。",
		LangJA: "ハッシュと共通鍵暗号は秘密鍵と対称暗号化を守ります。通常は SHA256 または SHA512 と AES256 です。",
	},
}
