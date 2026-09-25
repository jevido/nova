// Small enhancements; the page works fully without JavaScript.
const REPO = "jevido/nova";
const DL = `https://github.com/${REPO}/releases/latest/download/`;

// Recommend the download for the visitor's device.
const ua = navigator.userAgent;
const isAndroid = /Android/i.test(ua);
const isLinux = !isAndroid && /Linux|X11/i.test(ua);
const isMobile = /Android|iPhone|iPad|iPod|Mobile/i.test(ua);

const primary = document.querySelector("[data-dl]");
const label = primary.querySelector("[data-label]");
const sub = primary.querySelector("[data-sub]");

if (isAndroid) {
  primary.href = DL + "nova-android-arm64.apk";
  label.textContent = "Download for Android";
  sub.textContent = "APK · Android 5.0+";
  document.querySelector('[data-platform="android"]').classList.add("recommended");
  // Show Android first on phones.
  const platforms = document.querySelector(".platforms");
  platforms.prepend(document.querySelector('[data-platform="android"]'));
} else if (isLinux) {
  label.textContent = "Download for Linux";
  sub.textContent = "Quick install or package";
  document.querySelector('[data-platform="linux"]').classList.add("recommended");
}

// On a computer, offer a QR code to get the APK onto a phone.
if (!isMobile) document.querySelector("[data-qr]").hidden = false;

// Copy buttons.
document.querySelectorAll("[data-copy]").forEach((btn) => {
  btn.addEventListener("click", async () => {
    const text = document.querySelector(btn.dataset.copy).textContent.trim();
    try {
      await navigator.clipboard.writeText(text);
      btn.textContent = "Copied!";
      btn.classList.add("done");
    } catch {
      btn.textContent = "Press Ctrl+C";
      const range = document.createRange();
      range.selectNodeContents(document.querySelector(btn.dataset.copy));
      getSelection().removeAllRanges();
      getSelection().addRange(range);
    }
    setTimeout(() => {
      btn.textContent = "Copy";
      btn.classList.remove("done");
    }, 2000);
  });
});

// Show the latest version, or say so when nothing is released yet.
fetch(`https://api.github.com/repos/${REPO}/releases/latest`, { headers: { Accept: "application/vnd.github+json" } })
  .then((r) => (r.ok ? r.json() : r.status === 404 ? null : Promise.reject()))
  .then((rel) => {
    if (rel && rel.tag_name) {
      const date = new Date(rel.published_at).toLocaleDateString(undefined, { day: "numeric", month: "short", year: "numeric" });
      document.querySelector("[data-version]").textContent = `${rel.tag_name} is out · ${date}`;
      document.querySelector("[data-version-footer]").textContent = rel.tag_name;
    } else {
      document.querySelector("[data-version]").textContent = "First release coming soon";
      document.querySelector("[data-release-note]").innerHTML =
        `No release is published yet, so these links won't work just yet. In the meantime you can grab a build from the latest <a href="https://github.com/${REPO}/actions/workflows/ci.yml">CI run</a>.`;
    }
  })
  .catch(() => {});
