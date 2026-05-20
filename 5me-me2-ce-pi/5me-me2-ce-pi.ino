#include <Wire.h>
#include "unit_rolleri2c.hpp"

// AtomS3 PORT A (Grove): SDA=2, SCL=1
// #define SDA_PIN    2
// #define SCL_PIN    1

// ATOM MOTION PORTC
#define SDA_PIN    6
#define SCL_PIN    5
#define SAMPLE_MS  20  // サンプリング周期 [ms]

UnitRollerI2C roller;

void setup() {
    Serial.begin(115200);

    if (!roller.begin(&Wire, I2C_ADDR, SDA_PIN, SCL_PIN, 400000UL)) {
        Serial.println("ERROR: Roller485 が見つかりません");
        while (true) delay(1000);
    }

    roller.setOutput(0);
    roller.setMode(ROLLER_MODE_CURRENT);
    roller.setStallProtection(0);

    Serial.println("READY");
    Serial.println("# コマンド形式: I_pre[mA] I_step[mA] T_hold[ms]");
    Serial.println("# 計測中に \"stop\" を送信するとモーターを停止します");
}

void loop() {
    if (!Serial.available()) return;

    String line = Serial.readStringUntil('\n');
    line.trim();
    if (line.length() == 0) return;

    if (line == "stop") return;

    // "I_pre I_step T_hold" をパース
    int s1 = line.indexOf(' ');
    int s2 = line.indexOf(' ', s1 + 1);
    if (s1 < 0 || s2 < 0) {
        Serial.println("ERROR: コマンド形式: I_pre[mA] I_step[mA] T_hold[ms]");
        return;
    }

    int32_t  iPre  = line.substring(0, s1).toInt();
    int32_t  iStep = line.substring(s1 + 1, s2).toInt();
    uint32_t tHold = (uint32_t)line.substring(s2 + 1).toInt();

    // 計測開始
    Serial.println("time_ms,speed_rpm,current_ma");

    roller.setCurrent(iPre * 100);
    roller.setOutput(1);

    uint32_t startMs    = millis();
    uint32_t nextSample = startMs;
    bool     stepped    = false;

    while (true) {
        uint32_t now     = millis();
        uint32_t elapsed = now - startMs;

        // ステップ印加
        if (!stepped && elapsed >= tHold) {
            roller.setCurrent(iStep * 100);
            stepped = true;
        }

        // サンプリング
        if (now >= nextSample) {
            float speed   = roller.getSpeedReadback() / 100.0f;
            float current = roller.getCurrentReadback() / 100.0f;
            Serial.print(elapsed);
            Serial.print(',');
            Serial.print(speed, 1);
            Serial.print(',');
            Serial.println(current, 1);
            nextSample += SAMPLE_MS;
        }

        // シリアルから "stop" を受信したら中断
        if (Serial.available()) {
            String cmd = Serial.readStringUntil('\n');
            cmd.trim();
            if (cmd == "stop") {
                Serial.println("STOPPED");
                break;
            }
        }

        if (elapsed >= 2 * tHold) break;
    }

    roller.setOutput(0);
}
