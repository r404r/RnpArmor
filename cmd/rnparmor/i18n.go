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
		LangEN: "Menu-less About page: workspace tabs are hidden in this view.",
		LangZH: "无菜单 About 页面：该视图下工作区菜单已隐藏。",
		LangJA: "メニューなし About ページ: この画面ではワークスペースタブは非表示です。",
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
		LangEN: "Primary",
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
}
