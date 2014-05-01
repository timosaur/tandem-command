(function() {

  function TandemSpot(car) {
    var self = this;
    self.car = ko.observable(car || null);
    self.requests = ko.observableArray([]);

    self.request = function(car) {
      if (self.requests.indexOf(car) < 0) {
        if (car.request())
          car.request().requests.remove(car);
        car.request(self);
        self.requests.push(car);
      } else {
        self.requests.remove(car);
      }
    }

    self.park = function(car) {
      if (car) {
        if (car.spot())
          car.spot().car(null);
        car.spot(this);
        this.car(car);
      }
    }

    self.carHeight = ko.computed(function() {
      return 100/self.requests().length + '%';
    });
  }

  function Car(driver) {
    var self = this;
    self.driver = ko.observable(driver || null);
    self.spot = ko.observable();
    self.request = ko.observable();
  }

  function TandemViewModel() {
    var self = this;
    self.cars = ko.observableArray([
    ]);
    self.spots = ko.observableArray([
      new TandemSpot(),
      new TandemSpot()
    ]);
    self.car = ko.observable();

    self.selectCar = function(car) {
      if (car == self.car())
        self.car(null);
      else
        self.car(car);
    }
  };

  ko.applyBindings(new TandemViewModel());
})();
