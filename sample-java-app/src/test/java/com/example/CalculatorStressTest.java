package com.example;

import io.qameta.allure.*;
import org.junit.jupiter.api.*;
import org.junit.jupiter.params.ParameterizedTest;
import org.junit.jupiter.params.provider.CsvSource;
import java.util.Random;

import static org.junit.jupiter.api.Assertions.*;

/**
 * Stress test suite: generates a large volume of test results with binary attachments
 * so that the final allure-results.zip exceeds 100 MB.
 *
 * Each test attaches ~1 MB of pseudo-random bytes (screenshot + log).
 * Random bytes have near-zero compression ratio, keeping the zip large.
 */
@Epic("Calculator")
@Owner("stress-team")
class CalculatorStressTest {

    private Calculator calc;

    @BeforeEach
    void setUp() {
        calc = new Calculator();
    }

    @BeforeEach
    void checkPassRate(TestInfo testInfo) {
        int passRate = Integer.parseInt(System.getProperty("passRate", "100"));
        if (passRate < 100) {
            long seed = (long) testInfo.getDisplayName().hashCode() * 0xDEADBEEFL;
            if (new Random(seed).nextInt(100) >= passRate) {
                fail("Simulated failure: pass rate is " + passRate + "%");
            }
        }
    }

    @AfterEach
    void attachArtifacts(TestInfo testInfo) {
        captureScreenshot(testInfo.getDisplayName());
        captureExecutionLog(testInfo.getDisplayName());
    }

    @Attachment(value = "screenshot", type = "image/png")
    private byte[] captureScreenshot(String testName) {
        byte[] data = new byte[512 * 1024]; // 512 KB
        new Random((long) testName.hashCode() * 0xDEADBEEFL).nextBytes(data);
        return data;
    }

    @Attachment(value = "execution-log", type = "text/plain")
    private byte[] captureExecutionLog(String testName) {
        byte[] data = new byte[512 * 1024]; // 512 KB
        new Random((long) testName.hashCode() * 0xCAFEBABEL).nextBytes(data);
        return data;
    }

    // ── Addition stress ──────────────────────────────────────────────────────

    @Feature("Addition")
    @Story("Large range inputs")
    @Severity(SeverityLevel.NORMAL)
    @ParameterizedTest(name = "add({0}, {1}) = {2}")
    @CsvSource({
        "0, 0, 0", "1, 1, 2", "2, 3, 5", "5, 8, 13", "13, 21, 34",
        "34, 55, 89", "100, 200, 300", "999, 1, 1000", "-1, 1, 0", "-50, 50, 0",
        "1000, 2000, 3000", "500, 500, 1000", "123, 456, 579", "789, 321, 1110",
        "-100, -200, -300", "-999, 999, 0", "2147483, 1, 2147484", "1, -1, 0",
        "10, 90, 100", "25, 75, 100", "33, 67, 100", "49, 51, 100",
        "0, 1, 1", "1, 0, 1", "-1, 0, -1", "0, -1, -1",
        "100, 0, 100", "0, 100, 100", "-100, 0, -100", "0, -100, -100",
        "50, 50, 100", "25, 25, 50", "10, 10, 20", "5, 5, 10",
        "1000, 1000, 2000", "2000, 2000, 4000", "3000, 3000, 6000",
        "4000, 4000, 8000", "5000, 5000, 10000", "6000, 6000, 12000",
        "7000, 7000, 14000", "8000, 8000, 16000", "9000, 9000, 18000",
        "10000, 10000, 20000", "11000, 11000, 22000", "12000, 12000, 24000"
    })
    void stressAdd(int a, int b, int expected) {
        assertEquals(expected, calc.add(a, b));
    }

    // ── Subtraction stress ───────────────────────────────────────────────────

    @Feature("Subtraction")
    @Story("Large range inputs")
    @Severity(SeverityLevel.NORMAL)
    @ParameterizedTest(name = "subtract({0}, {1}) = {2}")
    @CsvSource({
        "10, 3, 7", "100, 50, 50", "1000, 999, 1", "0, 0, 0", "5, 5, 0",
        "-5, -3, -2", "-10, -10, 0", "1000, 1, 999", "500, 250, 250",
        "1, 1000, -999", "0, 100, -100", "0, -100, 100",
        "200, 100, 100", "300, 150, 150", "400, 200, 200", "500, 300, 200",
        "600, 400, 200", "700, 500, 200", "800, 600, 200", "900, 700, 200",
        "1000, 800, 200", "1100, 900, 200", "1200, 1000, 200", "1300, 1100, 200",
        "1400, 1200, 200", "1500, 1300, 200", "1600, 1400, 200", "1700, 1500, 200",
        "1800, 1600, 200", "1900, 1700, 200", "2000, 1800, 200", "2100, 1900, 200",
        "2200, 2000, 200", "2300, 2100, 200", "2400, 2200, 200", "2500, 2300, 200",
        "2600, 2400, 200", "2700, 2500, 200", "2800, 2600, 200", "2900, 2700, 200",
        "3000, 2800, 200", "3100, 2900, 200", "3200, 3000, 200", "3300, 3100, 200",
        "3400, 3200, 200", "3500, 3300, 200", "3600, 3400, 200", "3700, 3500, 200"
    })
    void stressSubtract(int a, int b, int expected) {
        assertEquals(expected, calc.subtract(a, b));
    }

    // ── Multiplication stress ────────────────────────────────────────────────

    @Feature("Multiplication")
    @Story("Large range inputs")
    @Severity(SeverityLevel.NORMAL)
    @ParameterizedTest(name = "multiply({0}, {1}) = {2}")
    @CsvSource({
        "0, 0, 0", "1, 1, 1", "2, 2, 4", "3, 3, 9", "4, 4, 16",
        "5, 5, 25", "6, 6, 36", "7, 7, 49", "8, 8, 64", "9, 9, 81",
        "10, 10, 100", "11, 11, 121", "12, 12, 144", "13, 13, 169",
        "14, 14, 196", "15, 15, 225", "16, 16, 256", "17, 17, 289",
        "18, 18, 324", "19, 19, 361", "20, 20, 400", "21, 21, 441",
        "22, 22, 484", "23, 23, 529", "24, 24, 576", "25, 25, 625",
        "2, 3, 6", "3, 4, 12", "4, 5, 20", "5, 6, 30", "6, 7, 42",
        "7, 8, 56", "8, 9, 72", "9, 10, 90", "10, 11, 110", "11, 12, 132",
        "-1, 1, -1", "-1, -1, 1", "-2, 3, -6", "-3, -4, 12", "0, 999, 0",
        "100, 100, 10000", "200, 200, 40000", "300, 300, 90000",
        "10, 100, 1000", "20, 100, 2000", "30, 100, 3000", "40, 100, 4000",
        "50, 100, 5000", "60, 100, 6000", "70, 100, 7000", "80, 100, 8000"
    })
    void stressMultiply(int a, int b, int expected) {
        assertEquals(expected, calc.multiply(a, b));
    }

    // ── Division stress ──────────────────────────────────────────────────────

    @Feature("Division")
    @Story("Large range inputs")
    @Severity(SeverityLevel.NORMAL)
    @ParameterizedTest(name = "divide({0}, {1}) = {2}")
    @CsvSource({
        "4, 2, 2.0",    "9, 3, 3.0",    "16, 4, 4.0",   "25, 5, 5.0",
        "36, 6, 6.0",   "49, 7, 7.0",   "64, 8, 8.0",   "81, 9, 9.0",
        "100, 10, 10.0","1000, 10, 100.0","1000, 100, 10.0",
        "1000, 1000, 1.0","500, 250, 2.0","500, 125, 4.0",
        "1, 2, 0.5",    "1, 4, 0.25",   "1, 5, 0.2",    "1, 10, 0.1",
        "3, 2, 1.5",    "5, 2, 2.5",    "7, 2, 3.5",    "9, 2, 4.5",
        "11, 2, 5.5",   "13, 2, 6.5",   "15, 2, 7.5",   "17, 2, 8.5",
        "19, 2, 9.5",   "21, 2, 10.5",  "23, 2, 11.5",  "25, 2, 12.5",
        "-4, 2, -2.0",  "-9, 3, -3.0",  "-16, 4, -4.0", "-25, 5, -5.0",
        "4, -2, -2.0",  "9, -3, -3.0",  "16, -4, -4.0", "-4, -2, 2.0",
        "200, 4, 50.0", "200, 5, 40.0", "200, 8, 25.0", "200, 10, 20.0",
        "200, 20, 10.0","200, 25, 8.0", "200, 40, 5.0", "200, 50, 4.0",
        "200, 100, 2.0","200, 200, 1.0"
    })
    void stressDivide(int a, int b, double expected) {
        assertEquals(expected, calc.divide(a, b), 0.0001);
    }

    // ── Prime stress ─────────────────────────────────────────────────────────

    @Feature("Prime check")
    @Story("Bulk prime validation")
    @Severity(SeverityLevel.NORMAL)
    @ParameterizedTest(name = "{0} is prime")
    @CsvSource({
        "2", "3", "5", "7", "11", "13", "17", "19", "23", "29",
        "31", "37", "41", "43", "47", "53", "59", "61", "67", "71",
        "73", "79", "83", "89", "97", "101", "103", "107", "109", "113",
        "127", "131", "137", "139", "149", "151", "157", "163", "167", "173",
        "179", "181", "191", "193", "197", "199", "211", "223", "227", "229"
    })
    void stressPrimes(int n) {
        assertTrue(calc.isPrime(n));
    }

    @Feature("Prime check")
    @Story("Bulk composite validation")
    @Severity(SeverityLevel.NORMAL)
    @ParameterizedTest(name = "{0} is not prime")
    @CsvSource({
        "1", "4", "6", "8", "9", "10", "12", "14", "15", "16",
        "18", "20", "21", "22", "24", "25", "26", "27", "28", "30",
        "32", "33", "34", "35", "36", "38", "39", "40", "42", "44",
        "45", "46", "48", "49", "50", "51", "52", "54", "55", "56",
        "57", "58", "60", "62", "63", "64", "65", "66", "68", "69"
    })
    void stressComposites(int n) {
        assertFalse(calc.isPrime(n));
    }

    // ── Modulo stress ────────────────────────────────────────────────────────

    @Feature("Modulo")
    @Story("Bulk modulo")
    @Severity(SeverityLevel.NORMAL)
    @ParameterizedTest(name = "{0} %% {1} = {2}")
    @CsvSource({
        "10, 3, 1", "10, 7, 3", "100, 7, 2", "100, 11, 1", "100, 13, 9",
        "50, 6, 2", "50, 7, 1", "50, 8, 2", "50, 9, 5", "50, 11, 6",
        "99, 10, 9", "98, 10, 8", "97, 10, 7", "96, 10, 6", "95, 10, 5",
        "94, 10, 4", "93, 10, 3", "92, 10, 2", "91, 10, 1", "90, 10, 0",
        "1000, 7, 6", "1000, 11, 10", "1000, 13, 12", "1000, 17, 14",
        "1000, 19, 12", "1000, 23, 11", "1000, 29, 14", "1000, 31, 8",
        "1000, 37, 1", "1000, 41, 22", "1000, 43, 31", "1000, 47, 12",
        "500, 7, 3", "500, 11, 6", "500, 13, 6", "500, 17, 7",
        "500, 19, 7", "500, 23, 17", "250, 7, 5", "250, 11, 8",
        "250, 13, 3", "250, 17, 13", "200, 7, 4", "200, 11, 2",
        "200, 13, 5", "200, 17, 14", "200, 19, 10", "200, 23, 16",
        "150, 7, 3", "150, 11, 7"
    })
    void stressModulo(int a, int b, int expected) {
        assertEquals(expected, calc.modulo(a, b));
    }

    // ── Square root stress ───────────────────────────────────────────────────

    @Feature("Square root")
    @Story("Bulk square root")
    @Severity(SeverityLevel.NORMAL)
    @ParameterizedTest(name = "sqrt({0}) = {1}")
    @CsvSource({
        "0.0, 0.0", "1.0, 1.0", "4.0, 2.0", "9.0, 3.0", "16.0, 4.0",
        "25.0, 5.0", "36.0, 6.0", "49.0, 7.0", "64.0, 8.0", "81.0, 9.0",
        "100.0, 10.0", "121.0, 11.0", "144.0, 12.0", "169.0, 13.0",
        "196.0, 14.0", "225.0, 15.0", "256.0, 16.0", "289.0, 17.0",
        "324.0, 18.0", "361.0, 19.0", "400.0, 20.0", "441.0, 21.0",
        "484.0, 22.0", "529.0, 23.0", "576.0, 24.0", "625.0, 25.0",
        "676.0, 26.0", "729.0, 27.0", "784.0, 28.0", "841.0, 29.0",
        "900.0, 30.0", "961.0, 31.0", "1024.0, 32.0", "1089.0, 33.0",
        "1156.0, 34.0", "1225.0, 35.0", "1296.0, 36.0", "1369.0, 37.0",
        "1444.0, 38.0", "1521.0, 39.0", "1600.0, 40.0", "1681.0, 41.0",
        "1764.0, 42.0", "1849.0, 43.0", "1936.0, 44.0", "2025.0, 45.0",
        "2116.0, 46.0", "2209.0, 47.0", "2304.0, 48.0", "2401.0, 49.0"
    })
    void stressSqrt(double input, double expected) {
        assertEquals(expected, calc.squareRoot(input), 0.001);
    }
}
