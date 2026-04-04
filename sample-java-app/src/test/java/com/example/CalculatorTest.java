package com.example;

import io.qameta.allure.*;
import org.junit.jupiter.api.*;
import org.junit.jupiter.params.ParameterizedTest;
import org.junit.jupiter.params.provider.CsvSource;

import java.util.Random;

import static org.junit.jupiter.api.Assertions.*;
import static org.junit.jupiter.api.Assumptions.*;

@Epic("Calculator")
@Owner("sample-team")
class CalculatorTest {

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
        byte[] data = new byte[512 * 1024]; // 512 KB of pseudo-random bytes
        new Random((long) testName.hashCode() * 0xDEADBEEFL).nextBytes(data);
        return data;
    }

    @Attachment(value = "execution-log", type = "text/plain")
    private byte[] captureExecutionLog(String testName) {
        byte[] data = new byte[512 * 1024]; // 512 KB of pseudo-random bytes
        new Random((long) testName.hashCode() * 0xCAFEBABEL).nextBytes(data);
        return data;
    }

    // ── Addition ─────────────────────────────────────────────────────────────

    @Feature("Addition")
    @Story("Positive operands")
    @Severity(SeverityLevel.CRITICAL)
    @Description("Two positive integers should be summed correctly.")
    @Test
    void addTwoPositiveNumbers() {
        int result = step_add(3, 4);
        step_assertEqual(7, result);
    }

    @Feature("Addition")
    @Story("Negative operands")
    @Severity(SeverityLevel.NORMAL)
    @Test
    void addNegativeNumbers() {
        int result = step_add(-5, -3);
        step_assertEqual(-8, result);
    }

    @Feature("Addition")
    @Story("Identity element")
    @Severity(SeverityLevel.MINOR)
    @Test
    void addZeroIsIdentity() {
        assertEquals(42, calc.add(42, 0));
        assertEquals(42, calc.add(0, 42));
    }

    @Feature("Addition")
    @Story("Positive operands")
    @Severity(SeverityLevel.NORMAL)
    @ParameterizedTest(name = "{0} + {1} = {2}")
    @CsvSource({
        "1,  1,  2",
        "10, 20, 30",
        "0,  0,  0",
        "-1, 1,  0",
        "100, -50, 50"
    })
    void addParameterized(int a, int b, int expected) {
        assertEquals(expected, calc.add(a, b));
    }

    // ── Subtraction ──────────────────────────────────────────────────────────

    @Feature("Subtraction")
    @Story("Positive operands")
    @Severity(SeverityLevel.CRITICAL)
    @Test
    void subtractPositiveNumbers() {
        assertEquals(3, calc.subtract(10, 7));
    }

    @Feature("Subtraction")
    @Story("Negative result")
    @Severity(SeverityLevel.NORMAL)
    @Test
    void subtractResultIsNegative() {
        assertEquals(-5, calc.subtract(3, 8));
    }

    // ── Multiplication ───────────────────────────────────────────────────────

    @Feature("Multiplication")
    @Story("Positive operands")
    @Severity(SeverityLevel.CRITICAL)
    @Test
    void multiplyTwoPositiveNumbers() {
        assertEquals(12, calc.multiply(3, 4));
    }

    @Feature("Multiplication")
    @Story("Zero absorber")
    @Severity(SeverityLevel.NORMAL)
    @Test
    void multiplyByZeroIsZero() {
        assertEquals(0, calc.multiply(999, 0));
    }

    @Feature("Multiplication")
    @Story("Negative operands")
    @Severity(SeverityLevel.NORMAL)
    @ParameterizedTest(name = "{0} * {1} = {2}")
    @CsvSource({
        "-2,  3, -6",
        "-2, -3,  6",
        " 0,  5,  0"
    })
    void multiplyParameterized(int a, int b, int expected) {
        assertEquals(expected, calc.multiply(a, b));
    }

    // ── Division ─────────────────────────────────────────────────────────────

    @Feature("Division")
    @Story("Positive operands")
    @Severity(SeverityLevel.CRITICAL)
    @Test
    void divideTwoPositiveNumbers() {
        assertEquals(2.5, calc.divide(5, 2), 0.001);
    }

    @Feature("Division")
    @Story("Divide by zero")
    @Severity(SeverityLevel.BLOCKER)
    @Description("Dividing by zero must throw ArithmeticException.")
    @Test
    void divideByZeroThrows() {
        ArithmeticException ex = assertThrows(
            ArithmeticException.class,
            () -> calc.divide(10, 0)
        );
        assertTrue(ex.getMessage().contains("zero"));
    }

    // ── Prime check ──────────────────────────────────────────────────────────

    @Feature("Prime check")
    @Story("Known primes")
    @Severity(SeverityLevel.NORMAL)
    @ParameterizedTest(name = "{0} is prime")
    @CsvSource({"2", "3", "5", "7", "11", "13", "97"})
    void knownPrimesAreRecognised(int n) {
        assertTrue(calc.isPrime(n));
    }

    @Feature("Prime check")
    @Story("Known composites")
    @Severity(SeverityLevel.NORMAL)
    @ParameterizedTest(name = "{0} is not prime")
    @CsvSource({"1", "4", "6", "8", "9", "15", "100"})
    void knownCompositesAreRejected(int n) {
        assertFalse(calc.isPrime(n));
    }

    // ── Power ────────────────────────────────────────────────────────────────

    @Feature("Power")
    @Story("Integer exponentiation")
    @Severity(SeverityLevel.CRITICAL)
    @Issue("CALC-42")
    @Description("2^10 should be 1024. Known off-by-one bug in power() - intentional defect.")
    @Test
    void powerOfTwo() {
        long result = calc.power(2, 10);
        assertEquals(1024L, result, "2^10 must equal 1024 (got " + result + " - off-by-one bug)");
    }

    @Feature("Power")
    @Story("Zero exponent")
    @Severity(SeverityLevel.NORMAL)
    @Test
    void powerZeroExponentIsOne() {
        assertEquals(1L, calc.power(5, 0));
    }

    @Feature("Power")
    @Story("Negative base")
    @Severity(SeverityLevel.NORMAL)
    @Issue("CALC-43")
    @Description("(-2)^3 should be -8. Off-by-one bug also affects negative bases.")
    @Test
    void powerNegativeBase() {
        long result = calc.power(-2, 3);
        assertEquals(-8L, result, "(-2)^3 must equal -8 (got " + result + " - off-by-one bug)");
    }

    // ── Square root ──────────────────────────────────────────────────────────

    @Feature("Square root")
    @Story("Positive input")
    @Severity(SeverityLevel.NORMAL)
    @Test
    void squareRootOfPositive() {
        assertEquals(3.0, calc.squareRoot(9.0), 0.001);
    }

    @Feature("Square root")
    @Story("Negative input")
    @Severity(SeverityLevel.BLOCKER)
    @Issue("CALC-44")
    @Description("squareRoot of a negative number must throw ArithmeticException. " +
                 "Currently returns NaN - missing validation, intentional defect.")
    @Test
    void squareRootOfNegativeShouldThrow() {
        assertThrows(ArithmeticException.class,
            () -> calc.squareRoot(-9.0),
            "squareRoot(-9) must throw ArithmeticException (currently returns NaN)");
    }

    // ── Modulo ───────────────────────────────────────────────────────────────

    @Feature("Modulo")
    @Story("Positive operands")
    @Severity(SeverityLevel.NORMAL)
    @Link(name = "Spec", url = "https://en.wikipedia.org/wiki/Modulo")
    @ParameterizedTest(name = "{0} %% {1} = {2}")
    @CsvSource({"10, 3, 1", "15, 5, 0", "7, 4, 3"})
    void moduloPositive(int a, int b, int expected) {
        assertEquals(expected, calc.modulo(a, b));
    }

    @Feature("Modulo")
    @Story("Divide by zero")
    @Severity(SeverityLevel.BLOCKER)
    @Test
    void moduloByZeroThrows() {
        assertThrows(ArithmeticException.class, () -> calc.modulo(10, 0));
    }

    // ── Disabled / skipped ───────────────────────────────────────────────────

    @Feature("Factorial")
    @Story("Positive input")
    @Severity(SeverityLevel.NORMAL)
    @Disabled("CALC-50: factorial() not yet implemented")
    @Test
    void factorialOfPositive() {
        // calc.factorial(5) == 120
        fail("Not implemented");
    }

    @Feature("Factorial")
    @Story("Zero input")
    @Severity(SeverityLevel.MINOR)
    @Disabled("CALC-50: factorial() not yet implemented")
    @Test
    void factorialOfZeroIsOne() {
        // calc.factorial(0) == 1
        fail("Not implemented");
    }

    @Feature("Logarithm")
    @Story("Base 10")
    @Severity(SeverityLevel.NORMAL)
    @Disabled("CALC-51: log10() not yet implemented")
    @Test
    void logarithmBase10() {
        // calc.log10(100) == 2.0
        fail("Not implemented");
    }

    @Feature("Square root")
    @Story("Environment assumption")
    @Severity(SeverityLevel.MINOR)
    @Description("Aborted: requires strict-math mode flag not set in CI.")
    @Test
    void squareRootRequiresStrictMath() {
        assumeTrue(System.getProperty("strictMath") != null,
            "Skipped: set -DstrictMath to enable this test");
        assertEquals(4.0, calc.squareRoot(16.0), 0.001);
    }

    // ── Allure step helpers ──────────────────────────────────────────────────

    @Step("Add {a} + {b}")
    private int step_add(int a, int b) {
        return calc.add(a, b);
    }

    @Step("Assert result equals {expected}")
    private void step_assertEqual(int expected, int actual) {
        assertEquals(expected, actual);
    }
}
