# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## リポジトリの目的

制御工学実験「5ME-ME2-CE-EXP01」。M5Stack AtomS3 + Unit-Roller485（BLDCサーボ、I2C接続）を使って、電流入力 → 角速度出力の伝達関数を同定し、フィードバック制御系を構築する。実験仕様の唯一の出所は `plan.md`。設計判断（コマンド形式、サンプリング周期、出力CSVスキーマなど）はまず `plan.md` と突き合わせる。

`.claude/` は `.gitignore` 済み（コミットされない）。`CLAUDE.md` は git 管理下。

## ディレクトリ構成

実験ごとにファームウェア（M5側）とホストツール（PC側）をペアで配置する：

```
5me-me2-ce-step/                    # 実験1: ステップ応答による伝達関数同定
├── 5me-me2-ce-step.ino             # M5 firmware
└── roller485-step/                 # PC host (Go)

5me-me2-ce-pi/                      # 実験2: PI制御による目標角速度ステップ追従
├── 5me-me2-ce-pi.ino               # M5 firmware
└── roller485-pi/                   # PC host (Go)
```

ファームウェア・ホストはUSBシリアル（115200 baud）でやり取りする。

## アーキテクチャ

### ファームウェア（`*.ino`）

AtomS3で動作するArduinoスケッチ1ファイル構成。`UnitRollerI2C`（M5の `unit_rolleri2c.hpp`）でRoller485を駆動する。

- ピン定義はATOM MOTIONのPORT C 前提（`SDA_PIN=6, SCL_PIN=5`）。AtomS3本体のGrove（PORT A）を使う場合は `SDA=2, SCL=1` にコメントを差し替える。
- 起動時に電流モード（`ROLLER_MODE_CURRENT`）に切り替え、ストール保護をOFFにする。
- メインループは1コマンド = 1計測セッション。
  - `step` 版: `I_pre I_step T_hold\n` → 電流を平衡 → ステップ。CSVは `time_ms,speed_rpm,current_ma`。
  - `pi` 版: `Kp Ki omega_pre omega_step T_hold\n` → PI制御で `omega_pre` に整定 → `omega_step` へ目標値ステップ。CSVは `time_ms,omega_ref_rpm,speed_rpm,current_cmd_ma,current_meas_ma`（コマンド電流と実測電流の両方）。
- 計測中は `SAMPLE_MS`（20 ms）ごとにシリアルへCSV行を出力。
- 計測中に `stop\n` を受信すると即座にモーター停止して中断する。
- `pi` 版の電流飽和は `I_MAX_MA`（1000 mA）。アンチワインドアップは積分クランプ（`integral` を `±I_MAX/|Ki|` に制限。`Ki=0` のときはクランプをスキップして純P制御）。
- API単位の罠：`setCurrent` はmA × 100の整数を要求し、`getSpeedReadback()` / `getCurrentReadback()` は実値 × 100を返す。スケーリングを変更するときは両方を一緒に直す。

### ホストツール（`roller485-*/`、Go）

PC側のインタラクティブCLI。シリアルポート選択 → 計測パラメータ入力 → AtomS3へコマンド送信 → CSVとPNGプロットを保存。

- `main.go`: `go.bug.st/serial` でポート列挙・オープン。読み取りは別goroutineで `port.Read` → 自前バッファに溜めて `\n` で分割し、`chan string` に流す（`bufio.Scanner` は使わない。過去にバッファサイズ起因のバグがあったため意図的に手書き）。SIGINTで `stop\n` を送信してからexitする（モーター放置を避けるため）。
- `plot.go`: 標準ライブラリのみでブレゼンハム描画する自前プロッタ。軸ラベル無し、外部プロットライブラリも入っていない。
  - `step` 版: 上半分=速度（青）、下半分=電流（赤）。
  - `pi` 版: 上半分=`omega_ref`（灰）+ `speed`（青）を同スケールで重ね描き、下半分=`current_cmd`（橙）+ `current_meas`（赤）を同スケールで重ね描き。
- `pi` 版 `main.go` は `Kp`/`Ki` を `promptFloat` で受ける（その他は `promptInt`）。送信フォーマットは `"%g %g %d %d %d\n"`。CSVは5列パース（`len(parts) == 5`）。
- CSV/PNG は実行ファイルと同じディレクトリに `YYYYMMDDhhmmss.csv` / `.png` で保存される。
- `go.mod` の `go 1.25.9` 指定に注意（Go 1.25系が必要）。

## よく使うコマンド

ホストツール（`roller485-step/` または `roller485-pi/` 配下で実行）：

```bash
# クロスコンパイル（Taskfile）。Windows/Linux/macOS用バイナリを dist/ に生成
task                  # 3プラットフォーム全部
task windows          # Windows amd64 のみ
task linux            # Linux amd64 のみ
task macos            # macOS arm64 のみ

# 開発中の直接実行（現プラットフォーム向け）
go run .
```

ファームウェア：Arduino IDE または `arduino-cli` で対応する `.ino`（`5me-me2-ce-step.ino` / `5me-me2-ce-pi.ino`）をAtomS3向けにビルド/書き込み。テスト・lintはなし（手動でシリアル経由の動作確認のみ）。

## 編集時の注意

- シリアルコマンド仕様とCSVヘッダはファームウェア・ホスト両方の前提（step版=3列、pi版=5列）。片方だけ変更しない。
- ファームウェアはCSVの1行目にヘッダ行を出すが、ホスト側 (`collectData`) は数値パースに失敗した行を黙って捨てる設計のため、ヘッダ行はCSVファイルにそのまま残る。読み取り側のスクリプトを書くときはこの仕様を意識する。
- step版とpi版で共通ロジックを共有するときは、現状コピペ運用。共通パッケージ化はまだしていないので、片方を直したらもう片方も直すか、共通化のリファクタを別途行う。
- 改行 LF / 文字コード UTF-8、半角カッコ・「、」「。」を使う（グローバル設定どおり）。
