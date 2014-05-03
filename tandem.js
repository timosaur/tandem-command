(function() {

  function TandemSpot(car) {
    var self = this;
    self.parked = ko.observable(car || null);
    self.requests = ko.observableArray([]);

    self.request = function(car) {
      if (!car || self.parked() == car) return;
      if (self.requests.indexOf(car) < 0) {
        if (car.request())
          car.request().requests.remove(car);
        car.request(self);
        self.requests.push(car);
      } else {
        car.request(null);
        self.requests.remove(car);
      }
    }

    self.carHeight = ko.computed(function() {
      var numCars = self.requests().length;
      if (self.parked()) numCars++;
      return 100/numCars + '%';
    });
  }

  function Car(driver) {
    var self = this;
    self.driver = ko.observable(driver || null);
    self.request = ko.observable();
    self.parked = ko.observable();

    self.time = ko.observable();

    self.park = function() {
      if (self.request() && !self.request().parked()) {
        if (self.parked() && self.parked() != "street") {
          self.parked().parked(null);
        }
        self.request().requests.remove(self);
        self.request().parked(self);
        self.parked(self.request());
        self.request(null);
      } else if (self.parked() == "street") {
        self.parked(null);
      } else if (self.parked()) {
        self.parked().parked(null);
        self.parked(null);
      } else {
        self.parked("street");
      }
    }
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

    self.time = ko.observable();
    self.schedule = function() {
      self.car().time(parseInt(self.time()));
      self.time(null);
    }

    self.selectCar = function(car) {
      if (car == self.car())
        self.car(null);
      else
        self.car(car);
    }
  };

  ko.applyBindings(new TandemViewModel());
})();
