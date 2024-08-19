var $ = mdui.$
function changeNightMode(buttonElement,isInitial) {

    function changeVisible(elementList) {
        elementList.each(function() {
            if ($(this).hasClass('mdui-hidden')) {
                $(this).removeClass('mdui-hidden');
            } else {
                $(this).addClass('mdui-hidden');
            }
        });
    }

    function toggleNightMode() {
        if ($("body").hasClass('mdui-theme-layout-dark')) {
            $("body").removeClass('mdui-theme-layout-dark');
            localStorage.setItem('isNightMode', false);
        } else {
            $("body").addClass('mdui-theme-layout-dark');
            localStorage.setItem('isNightMode', true);
        }
    }
    if (isInitial){
        const isNightMode = localStorage.getItem('isNightMode') === 'true';
        if (!isNightMode) {
           return;
        }
    }
    
    toggleNightMode();
    
    changeVisible(buttonElement.children());
}

const global_nightmode_btn = $("#global-nightmode-btn");
global_nightmode_btn.on("click", function() {
  changeNightMode(global_nightmode_btn,false);
});

changeNightMode(global_nightmode_btn,true);