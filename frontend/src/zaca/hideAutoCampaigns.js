// Zaca: casilla «Ocultar envíos automáticos» de la pantalla de campañas.
//
// Las bienvenidas automáticas generan una campaña nueva por cada grupo de
// altas (llevan la etiqueta AUTO_TAG) y saturan el listado. La casilla va
// marcada por defecto y pide al servidor que las excluya (GET/DELETE
// /api/campaigns?exclude_tag=...), de modo que el total y la paginación
// cuadran. Su estado se recuerda en el navegador.
const AUTO_TAG = 'bienvenida-auto';
const STORAGE_KEY = 'zaca.campaigns.hideAuto';

function loadHideAuto() {
  try {
    const v = localStorage.getItem(STORAGE_KEY);
    return v === null ? true : v === 'true';
  } catch (e) {
    return true;
  }
}

function saveHideAuto(v) {
  try {
    localStorage.setItem(STORAGE_KEY, String(v));
  } catch (e) {
    // Sin localStorage (modo privado, etc.): se queda el valor por defecto.
  }
}

export default {
  data() {
    return {
      hideAuto: loadHideAuto(),
    };
  },

  methods: {
    // Parámetros extra para las llamadas a /api/campaigns (listado y borrado por consulta).
    zacaExcludeParams() {
      return this.hideAuto ? { exclude_tag: AUTO_TAG } : {};
    },

    onHideAutoChange(v) {
      this.hideAuto = v;
      saveHideAuto(v);

      // El conjunto de resultados cambia: vuelta a la primera página y sin selección.
      this.queryParams.page = 1;
      this.bulk.checked = [];
      this.bulk.all = false;
      this.getCampaigns();
    },
  },
};
