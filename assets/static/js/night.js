(function nightMode() {
  let nightMode = localStorage.getItem('isNightMode') === 'true';
  function setNightMode() {
    if (nightMode) {
      document.body.classList.add("mdui-theme-layout-dark");
    } else {
      document.body.classList.remove("mdui-theme-layout-dark");
    }
  }
  function registerNightModeSwitchBtn() {
    const nightModeSwitchBtn = document.getElementById('switch-nightmode-btn');
    const iconSwitchToDayMode = nightModeSwitchBtn.children[0];
    const iconSwitchToNightMode = nightModeSwitchBtn.children[1];
    nightModeSwitchBtn.addEventListener('click', function () {
      nightMode = !nightMode;
      localStorage.setItem('isNightMode', nightMode);
      setNightMode();
      if (nightMode) {
        iconSwitchToDayMode.classList.remove('mdui-hidden');
        iconSwitchToNightMode.classList.add('mdui-hidden');
      } else {
        iconSwitchToDayMode.classList.add('mdui-hidden');
        iconSwitchToNightMode.classList.remove('mdui-hidden');
      }
    });
    setNightMode(); // best effort
  }
  if (document.readyState === "loading") {
    document.addEventListener('DOMContentLoaded', registerNightModeSwitchBtn);
  } else {
    registerNightModeSwitchBtn();
  }
  setNightMode();
})();