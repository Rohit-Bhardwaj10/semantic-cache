import requests
import time
import random
import json

BASE_URL = "http://localhost:8080"
# Default dev secret JWT for "default" tenant with "admin" role
# In a real setup, generate this using the internal/api/middleware logic
# For simulation, we'll use the "default" fallback if we don't have a token,
# but since Auth is enabled, we'll provide a mock one or rely on the fallback 
# (assuming middleware is in 'dev' mode)

QUERIES = [
    "What is the capital of France?",
    "Tell me about Paris capital city.", # Semantic match to above
    "How to use semantic cache with Go?",
    "Explain the architecture of a multi-tier cache.",
    "What is the population of Tokyo?",
    "How far is Mars from Earth?",
    "Who won the world cup in 2022?",
    "What is a Large Language Model?"
]

def simulate_traffic():
    print("Starting Semantic Cache Traffic Simulation...")
    print(f"Targeting: {BASE_URL}")
    
    session = requests.Session()
    session.headers.update({"Content-Type": "application/json"})
    
    # Optional: Mock JWT Token (we'll see if the server environment allows it)
    # session.headers.update({"Authorization": "Bearer YOUR_EXPIRING_TOKEN"})

    try:
        while True:
            # Pick a random query
            q = random.choice(QUERIES)
            
            # 70% chance to pick a "hot" query to force cache hits
            if random.random() < 0.7:
                q = QUERIES[0]
                
            print(f"--- Sending: {q}")
            start = time.time()
            try:
                # Regular Query
                response = session.post(f"{BASE_URL}/cache/query", json={"query": q})
                
                if response.status_code == 200:
                    data = response.json()
                    latency = time.time() - start
                    print(f"Result: {data['source']} | Latency: {latency:.4f}s | Hit: {data['hit']}")
                else:
                    print(f"Error {response.status_code}: {response.text}")
                    
                # Small chance to provide feedback
                if random.random() < 0.2:
                    rid = response.headers.get("X-Request-ID", "sim-unk")
                    session.post(f"{BASE_URL}/feedback", json={
                        "request_id": rid,
                        "correct": True
                    })

                # Small chance to use the Stream API instead
                if random.random() < 0.1:
                    print("--- Streaming Query...")
                    s_resp = session.get(f"{BASE_URL}/cache/stream", params={"q": q}, stream=True)
                    for line in s_resp.iter_lines():
                        if line:
                            print(f"Stream: {line.decode('utf-8')[:50]}...")
                            
            except Exception as e:
                print(f"Request failed: {e}")
                
            time.sleep(random.uniform(0.5, 2.0))
            
    except KeyboardInterrupt:
        print("\nSimulation stopped.")

if __name__ == "__main__":
    simulate_traffic()
