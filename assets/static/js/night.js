var $ = mdui.$
function changeNightMode(buttonElement) {
    function changeVisible(elementList) {
        elementList.each(function() {
            if ($(this).hasClass('mdui-hidden')) {
                $(this).removeClass('mdui-hidden');
            } else {
                $(this).addClass('mdui-hidden');
            }
        });
    }

    changeVisible(buttonElement.children());
    if ($("body").hasClass('mdui-theme-layout-dark')) {
        $("body").removeClass('mdui-theme-layout-dark');
        return;
    }
    $("body").addClass('mdui-theme-layout-dark');
    return;
}

const global_nightmode_btn = $("#global-nightmode-btn");
global_nightmode_btn.on("click", function() {
  changeNightMode(global_nightmode_btn);
});