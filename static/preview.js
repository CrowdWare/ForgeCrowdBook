const loadBtn = document.getElementById("load-preview");
if (loadBtn) {
  loadBtn.addEventListener("click", async () => {
    const sourceInput = document.getElementById("source-url");
    const previewArea = document.getElementById("preview-area");
    const submitBtn = document.getElementById("submit-btn");
    if (!sourceInput || !previewArea || !submitBtn) return;

    const url = sourceInput.value.trim();
    if (!url) {
      previewArea.textContent = "Source URL is required.";
      return;
    }

    const csrf = document.querySelector('meta[name="csrf-token"]')?.content ?? "";
    const form = new URLSearchParams();
    form.set("source_url", url);
    form.set("_csrf", csrf);
    const res = await fetch(loadBtn.dataset.url || "/dashboard/preview", {
      method: "POST",
      headers: { "Content-Type": "application/x-www-form-urlencoded" },
      body: form.toString()
    });

    if (!res.ok) {
      previewArea.textContent = "Content unavailable.";
      submitBtn.disabled = true;
      return;
    }

    previewArea.innerHTML = await res.text();
    submitBtn.disabled = false;
  });
}
