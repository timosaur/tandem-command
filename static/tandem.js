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
    self.suggestion = ko.observable();

    self.park = function(spot) {
      if (!self.parked() && spot) {
        self.parked(spot);
        if (spot !== "street") {
          spot.parked(self);
        }
      }
    }

    self.parkRequested = function() {
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

    self.unpark = function() {
      if (self.parked()) {
        if (self.parked() !== "street") {
          self.parked().parked(null);
        }
        self.parked(null);
      }
    }
  }

  function TandemViewModel(commandId) {
    var self = this;
    self.id = commandId || 0;

    self.name = ko.observable();

    self.cars = ko.observableArray([]);
    self.spots = ko.observableArray([]);
    self.car = ko.observable();

    self.park = function(street) {
      if (!self.car()) return;
      var spot = "street";
      if (!street) {
        spot = ko.utils.arrayFirst(self.spots(), function(spot) {
          return !spot.parked();
        });
      }
      self.car().park(spot);
    }

    self.unpark = function() {
      if (!self.car() || !self.car().parked()) return;
      if (self.car().parked() !== "street") {
        var spotLoc = self.spots.indexOf(self.car().parked());
        for (var i = spotLoc + 1; i < self.spots().length; i++) {
          var spot = self.spots()[i];
          if (spot.parked()) {
            console.log(self.car().driver(), "blocked");
            return;
          }
        }
      }
      self.car().unpark();
    }

    self.time = ko.observable();
    self.schedule = function() {
      self.car().time(parseInt(self.time()));
      self.time(null);
    }

    self.suggestSpot = function(cars, numSpots) {
      var numCars = cars.length;
      if (numSpots === 0) {
        ko.utils.arrayForEach(cars, function(car) {
          car.suggestion("s");
        });
      } else if (numSpots === 1) {
        if (numCars === 1) {
          cars[0].suggestion("i");
        } else {
          ko.utils.arrayForEach(cars, function(car) {
            car.suggestion("s/i");
          });
        }
      } else {
        var carsBefore = 0;
        var carsAfter = numCars - 1;
        ko.utils.arrayForEach(cars, function(car) {
          if (carsBefore - carsAfter < 0) {
            car.suggestion("s");
          } else {
            car.suggestion("i");
          }
          carsBefore += 1;
          carsAfter -= 1;
        });
      }
    }
    ko.computed(function() {
      self.cars.sort(function(l, r) {
        return l.time() == r.time() ? 0 : (l.time() < r.time() ? -1 : 1)
      });
      var targetCars = [];
      var numSpots = self.spots().filter(function(spot) { return !spot.parked(); }).length;
      ko.utils.arrayForEach(self.cars(), function(car) {
        if (!car.parked()) {
          targetCars.push(car);
        } else {
          car.suggestion("parked");
          if (car.parked() != "street") {
            self.suggestSpot(targetCars, numSpots);
            targetCars = [];
            numSpots = 0;
          }
        }
      });
      self.suggestSpot(targetCars, numSpots);
    });

    self.selectCar = function(car) {
      if (car == self.car())
        self.car(null);
      else
        self.car(car);
    }

    self.save = function() {
      console.log("save");
      var saveData = {};
      saveData['name'] = self.name();
      saveData['spots'] = self.spots().length;
      saveData['cars'] = [];
      ko.utils.arrayForEach(self.cars(), function(car) {
        var carData = {};
        carData['driver'] = car.driver();
        carData['time'] = car.time();
        if (car.parked() === "street") {
           carData['parked'] = "s";
        } else {
          var spotLoc = self.spots.indexOf(car.parked());
          if (spotLoc !== -1) carData['parked'] = spotLoc.toString();
        }
        saveData['cars'].push(carData);
      });
      $.ajax({
        url: "/save?id=" + self.id,
        method: "put",
        dataType: "json",
        contentType: "application/json; charset=utf-8",
        data: JSON.stringify(saveData)
      }).success(function(data) {
        console.log("saved", data);
      })
    }

    /* Load */
    $.ajax({
      url: "/save?id=" + self.id,
      method: "get",
      dataType: "json"
    }).success(function(data) {
      self.name(data['name']);
      var numSpots = data['spots'];
      console.log("saved spots:", numSpots);
      for (; numSpots>0; numSpots--) {
        self.spots.push(new TandemSpot());
      }
      ko.utils.arrayForEach(data['cars'], function(carData) {
        var car = new Car(carData.driver);
        if (carData.schedule !== "")
          car.time(carData.schedule);
        if (carData.parked) {
          console.log("park", carData.driver, carData.parked);
          if (carData.parked === "s") {
            car.parked("street");
          } else {
            var parkedSpot = self.spots()[parseInt(carData.parked)];
            car.parked(parkedSpot);
            parkedSpot.parked(car);
          }
        }
        console.log("loaded", car.driver());
        self.cars.push(car);
      });
    });

  };

  var window = this || (0, eval)('this');
  window.initTandem = function(id) {
    ko.applyBindings(new TandemViewModel(id));
  };

})();
