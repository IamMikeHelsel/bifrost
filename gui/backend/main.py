from fastapi import FastAPI  # type: ignore

app = FastAPI()


@app.get("/")
async def read_root() -> dict[str, str]:
    """Root endpoint for the Bifrost GUI Backend."""
    return {"message": "Hello from Bifrost GUI Backend"}


@app.get("/api/status")
async def get_status() -> dict[str, str]:
    """Returns the current status of the Bifrost GUI Backend."""
    return {"status": "running", "version": "0.1.0"}


# To run this application (from the gui/backend directory):
# uvicorn main:app --reload
#
# Then open your browser to http://127.0.0.1:8000/
# Or for the status endpoint: http://127.0.0.1:8000/api/status

if __name__ == "__main__":
    import uvicorn  # type: ignore

    # This allows running the app with "python main.py"
    # For Bazel's py_binary, this __main__ block will be executed.
    uvicorn.run(app, host="0.0.0.0", port=8000)
