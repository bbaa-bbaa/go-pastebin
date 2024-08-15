function changeNightMode(buttonElement) {
    var $ = mdui.$
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
