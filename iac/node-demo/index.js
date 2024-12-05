export const handler = async (event, context) => {
  // Create an interceptor
  const interceptor = new ConsoleInterceptor();

  // Start intercepting
  interceptor.startIntercept();

  // Normal logging will now be captured
  console.log('Starting Lambda execution');
  console.log("Invoked with", JSON.stringify(event, null, 2))
  console.error('Sample error log');

  // Get captured logs for potential logging or debugging
  const capturedLogs = interceptor.getLogContent();

  interceptor.stopIntercept();

  return {stdout: capturedLogs, result: ""}
}

class ConsoleInterceptor {
  constructor() {
    // Store original console methods
    this._originalLog = console.log;
    this._originalError = console.error;

    // Buffers to store intercepted logs
    this.logBuffer = [];
  }

  // Start intercepting console methods
  startIntercept() {
    // Override console.log
    console.log = (...args) => {
      // Convert all arguments to strings and join
      const message = args.map(arg =>
        typeof arg === 'object' ? JSON.stringify(arg) : String(arg)
      ).join(' ');

      // Store in log buffer
      this.logBuffer.push(message);

      // Call original log method to maintain standard output
      this._originalLog.apply(console, args);
    };

    // Override console.error
    console.error = (...args) => {
      // Convert all arguments to strings and join
      const message = args.map(arg =>
        typeof arg === 'object' ? JSON.stringify(arg) : String(arg)
      ).join(' ');

      // Store in error buffer
      this.logBuffer.push(`[ERROR] ${message}`);

      // Call original error method to maintain standard error output
      this._originalError.apply(console, args);
    };
  }

  // Stop intercepting and restore original methods
  stopIntercept() {
    console.log = this._originalLog;
    console.error = this._originalError;
  }

  // Get captured log content
  getLogContent() {
    return this.logBuffer.join('\n');
  }

  // Clear buffers
  clearBuffers() {
    this.logBuffer = [];
  }
}
