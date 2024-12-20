if ("serviceWorker" in navigator) {
  try {
    navigator.serviceWorker.addEventListener("message", event => {
      if (event.data.type === "update") {
        if (navigator.serviceWorker.controller) navigator.serviceWorker.controller.postMessage({ type: "notify-updated", id: event.data.id });
        mdui.snackbar({
          message: "检测到新版本，是否刷新？",
          buttonText: "更新",
          onButtonClick: () => {
            location.reload();
          },
          timeout: 0
        });
      }
    });
    navigator.serviceWorker
      .register("/sw.js", {
        scope: "/"
      })
      .then(registration => {
        if (registration.installing) {
          console.log("Service worker installing");
        } else if (registration.waiting) {
          console.log("Service worker installed");
        } else if (registration.active) {
          console.log("Service worker active");
        }
      });
  } catch (error) {
    console.error(`Registration failed with ${error}`);
  }
}
