from http.server import BaseHTTPRequestHandler, HTTPServer
import requests
import time
import json

class RequestHandler(BaseHTTPRequestHandler):
    def _set_headers(self):
        self.send_response(200)
        self.send_header('Content-type', 'text/plain')
        self.end_headers()

    def do_POST(self):
        pass
        # content_length = int(self.headers['Content-Length'])
        # post_data = self.rfile.read(content_length)

        # # json_data = json.loads(post_data)
        # # json_data['payload']['status'] = 'ok'

        # # print(json_data)

        # # Forward the received request to another URL
        # response = requests.post("http://127.0.0.1:8800/coding", data=post_data)

        # Set response headers
        self._set_headers()

        # # Send the response from the destination URL back to the client
        # self.wfile.write(response.content)

def run(server_class=HTTPServer, handler_class=RequestHandler, port=8080):
    server_address = ('', port)
    httpd = server_class(server_address, handler_class)
    print(f"Starting HTTP server on port {port}...")
    httpd.serve_forever()

if __name__ == "__main__":
    run()
