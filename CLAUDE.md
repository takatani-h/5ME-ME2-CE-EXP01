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

5me-me2-ce-pi/                      # 実験2: PI制御（roller485-step からコピーして派生中）
├── 5me-me2-ce-pi.ino               # M5 firmware
└── roller485-pi/                   # PC host (Go)
```

ファームウェア・ホストはUSBシリアル（115200 baud）でやり取りする。

## アーキテクチャ

### ファームウェア（`*.ino`）

AtomS3で動作するArduinoスケッチ1ファイル構成。`UnitRollerI2C`（M5の `unit_rolleri2c.hpp`）でRoller485を駆動する。

- ピン定義はATOM MOTIONのPORT C 前提（`SDA_PIN=6, SCL_PIN=5`）。AtomS3本体のGrove（PORT A）を使う場合は `SDA=2, SCL=1` にコメントを差し替える。
- 起動時に電流モード（`ROLLER_MODE_CURRENT`）に切り替え、ストール保護をOFFにする。
- メインループは1コマンド = 1計測セッション。`step` 版は `I_pre I_step T_hold\n` をパースして平衡 → ステップ → 計測を行う。
- 計測中は `SAMPLE_MS`（20 ms）ごとに `time_ms,speed_rpm,current_ma` をシリアルへ出力。
- 計測中に `stop\n` を受信すると即座にモーター停止して中断する。
- API単位の罠：`setCurrent` はmA × 100の整数を要求し、`getSpeedReadback()` / `getCurrentReadback()` は実値 × 100を返す。`plan.md` のスケーリングを変更するときは両方を一緒に直す。
- `5me-me2-ce-pi.ino` は現状 step版のコピーで、PI制御ロジックへの書き換えはこれから。

### ホストツール（`roller485-*/`、Go）

PC側のインタラクティブCLI。シリアルポート選択 → 計測パラメータ入力 → AtomS3へコマンド送信 → CSVとPNGプロットを保存。

- `main.go`: `go.bug.st/serial` でポート列挙・オープン。読み取りは別goroutineで `port.Read` → 自前バッファに溜めて `\n` で分割し、`chan string` に流す（`bufio.Scanner` は使わない。過去にバッファサイズ起因のバグがあったため意図的に手書き）。SIGINTで `stop\n` を送信してからexitする（モーター放置を避けるため）。
- `plot.go`: 標準ライブラリのみでブレゼンハム描画する自前プロッタ。上半分が速度（青）、下半分が電流（赤）、軸ラベル無し。外部プロットライブラリは入っていない。
- CSV/PNG は実行ファイルと同じディレクトリに `YYYYMMDDhhmmss.csv` / `.png` で保存される。
- `go.mod` の `go 1.25.9` 指定に注意（Go 1.25系が必要）。
- `roller485-pi/` は現状 `roller485-step/` のコピー（モジュール名とビルド出力名のみ書き換え済み）。PI制御用ロジックへの差し替えはこれから。

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

- `plan.md` のシリアルコマンド仕様とCSVヘッダ（`time_ms,speed_rpm,current_ma`）は、ファームウェア・ホスト両方の前提。片方だけ変更しない。
- ファームウェア側のCSVは1行目に `time_ms,speed_rpm,current_ma` を出すが、ホスト側 (`collectData`) は `len(parts) == 3` の数値パースに失敗した行を黙って捨てる設計のため、ヘッダ行はCSVファイルにそのまま残る。読み取り側のスクリプトを書くときはこの仕様を意識する。
- step版とpi版で共通ロジックを共有するときは、現状コピペ運用。共通パッケージ化はまだしていないので、片方を直したらもう片方も直すか、共通化のリファクタを別途行う。
- 改行 LF / 文字コード UTF-8、半角カッコ・「、」「。」を使う（グローバル設定どおり）。
