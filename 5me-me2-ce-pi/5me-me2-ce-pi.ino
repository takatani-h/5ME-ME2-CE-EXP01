#include <Wire.h>
#include "unit_rolleri2c.hpp"

// AtomS3 PORT A (Grove): SDA=2, SCL=1
// #define SDA_PIN    2
// #define SCL_PIN    1

// ATOM MOTION PORTC
#define SDA_PIN    6
#define SCL_PIN    5
#define SAMPLE_MS  20  // 制御周期 [ms]

constexpr float I_MAX_MA = 1000.0f;  // 電流飽和上限 [mA]

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
    Serial.println("# コマンド形式: Kp Ki omega_pre[RPM] omega_step[RPM] T_hold[ms]");
    Serial.println("# 計測中に \"stop\" を送信するとモーターを停止します");
}

void loop() {
    if (!Serial.available()) return;

    String line = Serial.readStringUntil('\n');
    line.trim();
    if (line.length() == 0) return;

    if (line == "stop") return;

    // "Kp Ki omega_pre omega_step T_hold" をパース
    int s1 = line.indexOf(' ');
    int s2 = line.indexOf(' ', s1 + 1);
    int s3 = line.indexOf(' ', s2 + 1);
    int s4 = line.indexOf(' ', s3 + 1);
    if (s1 < 0 || s2 < 0 || s3 < 0 || s4 < 0) {
        Serial.println("ERROR: コマンド形式: Kp Ki omega_pre[RPM] omega_step[RPM] T_hold[ms]");
        return;
    }

    float    kp        = line.substring(0, s1).toFloat();
    float    ki        = line.substring(s1 + 1, s2).toFloat();
    int32_t  omegaPre  = line.substring(s2 + 1, s3).toInt();
    int32_t  omegaStep = line.substring(s3 + 1, s4).toInt();
    uint32_t tHold     = (uint32_t)line.substring(s4 + 1).toInt();

    // 計測開始
    Serial.println("time_ms,omega_ref_rpm,speed_rpm,current_cmd_ma,current_meas_ma");

    roller.setCurrent(0);
    roller.setOutput(1);

    const float dt = SAMPLE_MS / 1000.0f;
    float integral = 0.0f;

    uint32_t startMs    = millis();
    uint32_t nextSample = startMs;

    while (true) {
        uint32_t now     = millis();
        uint32_t elapsed = now - startMs;

        if (now >= nextSample) {
            float omegaRef = (elapsed < tHold) ? (float)omegaPre : (float)omegaStep;
            float speed    = roller.getSpeedReadback() / 100.0f;
            float error    = omegaRef - speed;

            integral += error * dt;
            // 積分クランプ式アンチワインドアップ
            if (ki != 0.0f) {
                float intLimit = I_MAX_MA / fabsf(ki);
                if (integral >  intLimit) integral =  intLimit;
                if (integral < -intLimit) integral = -intLimit;
            }

            float u = kp * error + ki * integral;
            if (u >  I_MAX_MA) u =  I_MAX_MA;
            if (u < -I_MAX_MA) u = -I_MAX_MA;

            roller.setCurrent((int32_t)(u * 100));

            float measured = roller.getCurrentReadback() / 100.0f;

            Serial.print(elapsed);
            Serial.print(',');
            Serial.print(omegaRef, 1);
            Serial.print(',');
            Serial.print(speed, 1);
            Serial.print(',');
            Serial.print(u, 1);
            Serial.print(',');
            Serial.println(measured, 1);

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
