package com.example;

/**
 * Simple calculator — the class under test for the sample Allure report.
 */
public class Calculator {

    public int add(int a, int b) {
        return a + b;
    }

    public int subtract(int a, int b) {
        return a - b;
    }

    public int multiply(int a, int b) {
        return a * b;
    }

    public double divide(int a, int b) {
        if (b == 0) throw new ArithmeticException("Cannot divide by zero");
        return (double) a / b;
    }

    public boolean isPrime(int n) {
        if (n < 2) return false;
        for (int i = 2; i * i <= n; i++) {
            if (n % i == 0) return false;
        }
        return true;
    }

    /** BUG: uses exp-1 instead of exp — intentional defect for demo. */
    public long power(int base, int exp) {
        if (exp == 0) return 1;
        long result = 1;
        for (int i = 0; i < exp - 1; i++) {  // off-by-one: should be i < exp
            result *= base;
        }
        return result;
    }

    /** BUG: does not guard against negative input — intentional defect for demo. */
    public double squareRoot(double n) {
        return Math.sqrt(n);  // returns NaN for negatives instead of throwing
    }

    public int modulo(int a, int b) {
        if (b == 0) throw new ArithmeticException("Cannot modulo by zero");
        return a % b;
    }
}
