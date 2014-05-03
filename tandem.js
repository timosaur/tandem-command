(function() {
  var suggestions = {
    '': {
      0: 's',
      1: 'i',
      2: 'i'
    },
    '0': {
      1: 's/i',
      2: 'i'
    },
    '1': {
      0: 'o',
      2: 's'
    },
    '2': {
      0: 's/o',
      1: 's'
    },
    '01': {
      2: 'w'
    },
    '02': {
      1: 'o'
    },
    '10': {
      2: 's'
    },
    '12': {
      0: 'o'
    },
    '20': {
      1: 's'
    },
    '20s': {
      1: 'o'
    },
    '21': {
      0: 'o'
    }

  }

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
    ko.utils.arrayForEach(self.cars(), function(car) {
      car.parked.subscribe(function () {
        if (self.parkedCars.indexOf(car) < 0) {
          self.parkedCars.push(car);
        } else {
          self.parkedCars.remove(car);
        }
      });
    });

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

    self.parkedCars = ko.observableArray([]);
    self.orderedCars = ko.computed(function() {
      self.cars.sort(function(l, r) {
        return l.time() == r.time() ? 0 : (l.time() < r.time() ? -1 : 1)
      });
      var parkedorder = [];
      ko.utils.arrayForEach(self.parkedCars(), function(car) {
        var i = self.cars.indexOf(car);
        if (car.parked() == 'street' && (i == 0 && self.cars.length == 2)) {
          i += 's';
        }
        parkedorder.push(i);
      });
      var parker = parkedorder.join('');
      console.log("parker", parker);
      var oi = 0;
      for (var i in self.cars()) {
        var car = self.cars()[i];
        if (self.parkedCars.indexOf(car) >= 0) {
          car.suggestion("");
        } else {
          car.suggestion(suggestions[parker][i]);
        }
      }
    });

    self.selectCar = function(car) {
      if (car == self.car())
        self.car(null);
      else
        self.car(car);
    }
  };

  ko.applyBindings(new TandemViewModel());
})();
