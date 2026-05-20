# 実験1: ステップ電流応答による伝達関数同定

## 概要

電流入力 $I$ [mA] から角速度出力 $\omega$ [RPM] への1次系モデル

$$
\frac{\omega(s)}{I(s)} = \frac{K}{\tau s + 1}
$$

の DC ゲイン $K$ と時定数 $\tau$ を、ステップ電流応答から同定するための計測プログラム。AtomS3 で Roller485 を電流モード駆動し、平衡電流 $I_{\text{pre}}$ で整定したあと $I_{\text{step}}$ にステップ印加。前後の角速度時系列を CSV としてシリアル経由で PC に流す。

理論背景・パラメータ算出手順は [`../plan.md`](../plan.md) を参照。

## ディレクトリ構成

- `5me-me2-ce-step.ino` — AtomS3 ファームウェア
- `roller485-step/` — PC ホストツール (Go)

## ハードウェア

- M5Stack AtomS3 + ATOM MOTION + Unit-Roller485 (I2C)
- 接続: ATOM MOTION PORT C (`SDA=6, SCL=5`)。AtomS3 本体 Grove (PORT A, `SDA=2, SCL=1`) を使う場合はファームウェア冒頭のピン定義をコメント差し替え。

## シリアルプロトコル

- ボーレート: 115200
- コマンド形式 (改行で送信):

  ```
  I_pre[mA] I_step[mA] T_hold[ms]
  ```

  例: `200 300 5000` → 200 mA で 5000 ms 平衡 → 300 mA へステップして 5000 ms 計測。
- 計測中 `stop\n` を送るとモータ停止 + 中断。
- CSV (シリアル → ホスト):

  ```
  time_ms,speed_rpm,current_ma
  0,150.3,200.1
  20,151.0,199.8
  ...
  ```

  サンプリング周期は 20 ms。

## 使い方

1. ファームウェア書き込み: Arduino IDE で `5me-me2-ce-step.ino` を AtomS3 に書き込む。
2. ホスト起動:

   ```bash
   cd roller485-step
   go run .
   ```

   または `task` でクロスビルドした `dist/` 配下のバイナリを実行。
3. シリアルポート選択 → `I_pre [mA]` / `I_step [mA]` / `T_hold [ms]` を入力。
4. 実行ファイルと同じディレクトリに `YYYYMMDDhhmmss.csv` と `.png` が自動保存される。`Ctrl-C` または `stop` 入力で中断可能。

## ビルド (ホスト)

```bash
cd roller485-step
go run .                       # 開発実行 (現プラットフォーム向け)
task                           # Win/Linux/macOS 同時クロスビルド → dist/
task windows / linux / macos   # プラットフォーム個別
```

Go 1.25 系が必要 (`go.mod` の `go 1.25.9` 指定)。

## 出力

- `YYYYMMDDhhmmss.csv` — 計測データ。1行目にヘッダ `time_ms,speed_rpm,current_ma` が残る（ホスト側は数値パース失敗行を黙って捨てる設計）。
- `YYYYMMDDhhmmss.png` — プロット。
  - 上半分: 速度 (青)
  - 下半分: 電流 (赤)
  - 軸ラベル無し、自動スケール。

## パラメータ算出

定常値 $\omega_{ss}$、DC ゲイン $K = \omega_{ss} / I_{\text{step}}$、時定数 $\tau$ (= $\omega(t) = 0.632\,\omega_{ss}$ となる時刻) の求め方は [`../plan.md`](../plan.md) 実験1節を参照。
