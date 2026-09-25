// Small enhancements; the page works fully without JavaScript.
const REPO = "jevido/nova";
const DL = `https://github.com/${REPO}/releases/latest/download/`;

// Recommend the download for the visitor's device.
const ua = navigator.userAgent;
// iPadOS reports itself as a Mac; touch support gives it away.
const isIOS = /iPhone|iPad|iPod/i.test(ua) || (/Macintosh/i.test(ua) && navigator.maxTouchPoints > 1);
const isAndroid = /Android/i.test(ua);
const isWindows = /Windows/i.test(ua);
const isLinux = !isAndroid && /Linux|X11/i.test(ua);
const isMobile = isIOS || /Android|Mobile/i.test(ua);

const primary = document.querySelector("[data-dl]");
const label = primary.querySelector("[data-label]");
const sub = primary.querySelector("[data-sub]");

function recommend(platform) {
  const card = document.querySelector(`[data-platform="${platform}"]`);
  card.classList.add("recommended");
  // The visitor's platform goes first.
  document.querySelector(".platforms").prepend(card);
}

if (isAndroid) {
  primary.href = DL + "nova-android-arm64.apk";
  label.textContent = "Download for Android";
  sub.textContent = "APK · Android 5.0+";
  recommend("android");
} else if (isIOS) {
  // An IPA can't be installed by tapping it, so point at the steps instead.
  primary.href = "#ios";
  label.textContent = "Get Nova for iPhone & iPad";
  sub.textContent = "Sideload with AltStore";
  recommend("ios");
} else if (isWindows) {
  primary.href = DL + "nova-windows-amd64-setup.exe";
  label.textContent = "Download for Windows";
  sub.textContent = "Installer · Windows 10 & 11";
  recommend("windows");
} else if (isLinux) {
  label.textContent = "Download for Linux";
  sub.textContent = "Quick install or package";
  recommend("linux");
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
      document.querySelector("[data-version]").textContent = `Version ${rel.tag_name.replace(/^v/, "")} · ${date} · open source`;
      document.querySelector("[data-version-footer]").textContent = rel.tag_name;
    } else {
      document.querySelector("[data-version]").textContent = "First release coming soon";
      document.querySelector("[data-release-note]").innerHTML =
        `No release is published yet, so these links won't work just yet. Check back soon, or watch the project on <a href="https://github.com/${REPO}">GitHub</a>.`;
    }
  })
  .catch(() => {});
