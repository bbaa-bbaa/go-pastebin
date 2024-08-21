(function nightMode() {
  let nightMode = localStorage.getItem("isNightMode") === "true";
  function setMode() {
    if (nightMode) {
      if (!document.body.classList.contains("mdui-theme-layout-dark")) {
        document.body.classList.add("mdui-theme-layout-dark");
      }
    } else {
      document.body.classList.remove("mdui-theme-layout-dark");
    }
    document.cookie = `color_scheme=${nightMode ? "dark" : "light"};path=/;max-age=31536000`;
  }
  function registerNightModeSwitchBtn() {
    const nightModeSwitchBtn = document.getElementById("switch-nightmode-btn");
    const iconSwitchToDayMode = nightModeSwitchBtn.children[0];
    const iconSwitchToNightMode = nightModeSwitchBtn.children[1];
    function setIcon() {
      if (nightMode) {
        iconSwitchToDayMode.style.removeProperty("display");
        iconSwitchToNightMode.style.display = "none";
      } else {
        iconSwitchToDayMode.style.display = "none";
        iconSwitchToNightMode.style.removeProperty("display");
      }
    }
    nightModeSwitchBtn.addEventListener("click", function () {
      nightMode = !nightMode;
      localStorage.setItem("isNightMode", nightMode);
      setMode();
      setIcon();
    });
    setMode(); // best effort
    setIcon();
  }
  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", registerNightModeSwitchBtn);
    setMode();
  } else {
    registerNightModeSwitchBtn();
  }
})();
