function setToTop() {
    $('body').append('<div id="toTop" title="回到顶部"><span class="fa fa-arrow-circle-up"></span></div>');
    $(window).scroll(function() {
        if ($(this).scrollTop()) {
            $('#toTop').fadeIn();
        } else {
            $('#toTop').fadeOut();
        }
    });

    $("#toTop").click(function () {
        //html works for FFX but not Chrome
        //body works for Chrome but not FFX
        //This strange selector seems to work universally
        $("html, body").animate({scrollTop: 0}, 200);
    });
}

$(document).ready(function(){
    setToTop();
});