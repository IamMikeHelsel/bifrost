from fastapi import FastAPI

app = FastAPI()


@app.get("/")
async def read_root():
    return {"message": "Hello from Bifrost GUI Backend"}


@app.get("/api/status")
async def get_status():
    return {"status": "running", "version": "0.1.0"}


# To run this application (from the gui/backend directory):
# uvicorn main:app --reload
#
# Then open your browser to http://127.0.0.1:8000/
# Or for the status endpoint: http://127.0.0.1:8000/api/status

if __name__ == "__main__":
    import uvicorn

    # This allows running the app with "python main.py"
    # For Bazel's py_binary, this __main__ block will be executed.
    uvicorn.run(app, host="0.0.0.0", port=8000)
