from http.server import BaseHTTPRequestHandler, HTTPServer
import os
import json
import time
from prometheus_client import Counter, Histogram, generate_latest
from urllib.parse import urlparse, parse_qs

# Метрики
REQUEST_COUNT = Counter('hello_requests_total', 'Total requests', ['method', 'endpoint'])
REQUEST_DURATION = Histogram('hello_request_duration_seconds', 'Request duration')

class Handler(BaseHTTPRequestHandler):
    def do_GET(self):
        start_time = time.time()
        path = urlparse(self.path).path
        
        if path == "/metrics":
            self.send_response(200)
            self.send_header("Content-type", "text/plain")
            self.end_headers()
            self.wfile.write(generate_latest())
        elif path == "/healthz":
            self.send_response(200)
            self.send_header("Content-type", "application/json")
            self.end_headers()
            response = {"status": "healthy"}
            self.wfile.write(json.dumps(response).encode())
        else:
            self.send_response(200)
            self.send_header("Content-type", "application/json")
            self.end_headers()
            response = {"message": "Hello from Python Knative function!"}
            self.wfile.write(json.dumps(response).encode())
        
        # Обновляем метрики
        REQUEST_COUNT.labels(method="GET", endpoint=path).inc()
        REQUEST_DURATION.observe(time.time() - start_time)

if __name__ == "__main__":
    port = int(os.environ.get("PORT", 8080))
    HTTPServer(("", port), Handler).serve_forever()
