if ("serviceWorker" in navigator) {
  try {
    navigator.serviceWorker.addEventListener("message", event => {
      if (event.source && event.data.msgseq) {
        event.source.postMessage({ msgseq: event.data.msgseq, msgid: event.data.msgid });
      }
      if (event.data.type === "update") {
        sessionStorage.setItem("sw-update", "true");
        mdui.snackbar({
          message: "检测到新版本，是否刷新？",
          buttonText: "更新",
          onButtonClick: () => {
            location.reload();
          },
          timeout: 0
        });
      }
      if (event.data.type === "up-to-date" && sessionStorage.getItem("sw-update") === "true") {
        sessionStorage.removeItem("sw-update");
        mdui.snackbar({
          message: "已更新至最新版本",
          timeout: 3000
        });
      }
    });
    navigator.serviceWorker.ready.then(registration => {
      if (registration.active) {
        registration.active.postMessage({ type: "check-update" });
      } else {
        setTimeout(() => {
          if (navigator.serviceWorker.controller) navigator.serviceWorker.controller.postMessage({ type: "check-update" });
        }, 1000);
      }
    });
    navigator.serviceWorker.register("/sw.js", {
      scope: "/"
    });
  } catch (error) {
    console.error(`Registration failed with ${error}`);
  }
}
