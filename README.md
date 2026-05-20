# 5ME-ME2-CE-EXP01

制御工学実験。M5Stack AtomS3 + Unit-Roller485 (BLDCサーボ、I2C接続) を使い、電流入力 → 角速度出力のモデル同定と PI 制御による追従実験を行う。

## ハードウェア

- M5Stack AtomS3 (制御コントローラ)
- ATOM MOTION (拡張ベース、PORT C 経由で Roller485 を接続)
- Unit-Roller485 (BLDC サーボ + ドライバ、I2C アドレス `I2C_ADDR`)
- ピン: ATOM MOTION PORT C 経由で `SDA=6, SCL=5`
- AtomS3 のピンマップは [`AtomS3_PinMap.jpg`](AtomS3_PinMap.jpg) を参照
- Roller485 の I2C プロトコル詳細は [`Unit-Roller485-I2C-Protocol-EN.pdf`](Unit-Roller485-I2C-Protocol-EN.pdf) を参照
- 物理マウントは [`base_mount_set v0.3mf`](<base_mount_set v0.3mf>) (3D プリンタ用)

## ディレクトリ構成

| パス | 内容 |
| --- | --- |
| [`5me-me2-ce-step/`](5me-me2-ce-step/) | 実験1: ステップ電流応答による伝達関数同定 (AtomS3 ファームウェア + PC ホスト) |
| [`5me-me2-ce-pi/`](5me-me2-ce-pi/) | 実験2: PI制御による目標速度ステップ追従 (AtomS3 ファームウェア + PC ホスト) |
| [`plan.md`](plan.md) | 実験計画書 (理論・パラメータ算出手順) |

各実験ディレクトリの使い方は配下の `README.md` を参照:

- [実験1 README](5me-me2-ce-step/README.md)
- [実験2 README](5me-me2-ce-pi/README.md)

## 共通仕様

- ホストツールは Go 1.25 系で実装、シリアル 115200 baud で AtomS3 とやり取りする。
- ファームウェアは Arduino IDE で `*.ino` を書き込む 1ファイル構成。
- 計測結果は実行ファイルと同じディレクトリに `YYYYMMDDhhmmss.csv` / `.png` で自動保存。
- 計測中 `stop\n` をシリアル送信、または `Ctrl-C` でモータ停止して中断。

## クイックスタート

例として実験1を動かす場合:

```bash
# 1. Arduino IDE で 5me-me2-ce-step/5me-me2-ce-step.ino を AtomS3 に書き込み
# 2. ホストを起動
cd 5me-me2-ce-step/roller485-step
go run .
# シリアルポート選択 → I_pre / I_step / T_hold を入力 → CSV と PNG が保存される
```

実験2 (PI制御) も同じ要領。プロトコル詳細・計測例は各 README を参照。
