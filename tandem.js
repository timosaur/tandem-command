(function() {
  $('.car').click(function() {
    $(this).toggleClass('parked');
  });
  $('.car').hover(function() {
    $(this).toggleClass('requested');
  });
})();
